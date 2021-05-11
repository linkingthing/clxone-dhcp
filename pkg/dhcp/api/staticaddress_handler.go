package api

import (
	"fmt"
	"sort"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	//"github.com/linkingthing/clxone-dhcp/pkg/eventbus"
)

type StaticAddressHandler struct {
}

func NewStaticAddressHandler() *StaticAddressHandler {
	return &StaticAddressHandler{}
}

func (s *StaticAddressHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	staticAddress := ctx.Resource.(*resource.StaticAddress)
	if err := staticAddress.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create static address params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkMacOrIpInUsed(tx, subnet.GetID(), staticAddress.HwAddress,
			staticAddress.IpAddress, true); err != nil {
			return err
		}

		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnetIfCanCreateDynamicPool(subnet); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet.Ipnet, staticAddress.IpAddress) == false {
			return fmt.Errorf("static address ipaddress %s not belongs to subnet %s",
				staticAddress.IpAddress, subnet.Subnet)
		}

		if pdpool, conflict, err := checkIPConflictWithSubnetPDPool(tx, subnet.GetID(),
			staticAddress.IpAddress); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("static address ipaddress %s conflicts with pdpool %s in subnet %s",
				staticAddress.IpAddress, pdpool, subnet.GetID())
		}

		if conflictPool, conflict, err := checkPoolConflictWithSubnetPool(tx, subnet.GetID(),
			&resource.Pool{BeginAddress: staticAddress.IpAddress, EndAddress: staticAddress.IpAddress}); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("static address ipaddress %s conflicts with pool %s in subnet %s",
				staticAddress.IpAddress, conflictPool.String(), subnet.GetID())
		}

		if _, err := tx.Update(resource.TableSubnet, map[string]interface{}{
			"capacity": subnet.Capacity + 1,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		staticAddress.Capacity = 1
		staticAddress.Subnet = subnet.GetID()
		_, err := tx.Insert(staticAddress)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create static address with mac %s failed: %s", staticAddress.HwAddress, err.Error()))
	}

	//eventbus.PublishResourceCreateEvent(staticAddress)
	return staticAddress, nil
}

func checkMacOrIpInUsed(tx restdb.Transaction, subnetId, mac, ip string, isStatic bool) error {
	var count int64
	var err error
	if isStatic {
		count, err = tx.CountEx(resource.TableStaticAddress,
			"select count(*) from gr_static_address where subnet = $1 and (hw_address = $2 or ip_address = $3)",
			subnetId, mac, ip)
	} else {
		count, err = tx.CountEx(resource.TableStaticAddress,
			"select count(*) from gr_static_address where hw_address = $1 or ip_address = $2",
			mac, ip)
	}
	if err != nil {
		return fmt.Errorf("check static address %s-%s with subnet %s exists in db failed: %s",
			mac, ip, subnetId, err.Error())
	} else if count != 0 {
		return fmt.Errorf("static address exists with subnet %s and mac %s or ip %s", subnetId, mac, ip)
	}

	if isStatic {
		count, err = tx.CountEx(resource.TableReservation,
			"select count(*) from gr_reservation where hw_address = $1 or ip_address = $2",
			mac, ip)
	} else {
		count, err = tx.CountEx(resource.TableReservation,
			"select count(*) from gr_reservation where subnet = $1 and (hw_address = $2 or ip_address = $3)",
			subnetId, mac, ip)
	}

	if err != nil {
		return fmt.Errorf("check reservation %s-%s with subnet %s exists in db failed: %s",
			mac, ip, subnetId, err.Error())
	} else if count != 0 {
		return fmt.Errorf("reservation exists with subnet %s and mac %s or ip %s", subnetId, mac, ip)
	}

	return nil
}

func (s *StaticAddressHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var staticAddresss resource.StaticAddresses
	if err := db.GetResources(map[string]interface{}{"subnet": subnetID}, &staticAddresss); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list static addresss with subnet %s from db failed: %s", subnetID, err.Error()))
	}

	sort.Sort(staticAddresss)
	return staticAddresss, nil
}

func (s *StaticAddressHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	staticAddressID := ctx.Resource.GetID()
	var staticAddresss []*resource.StaticAddress
	staticAddress, err := restdb.GetResourceWithID(db.GetDB(), staticAddressID, &staticAddresss)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get static address %s with subnet %s from db failed: %s",
				staticAddressID, subnetID, err.Error()))
	}

	return staticAddress.(restresource.Resource), nil
}

func setStaticAddressFromDB(tx restdb.Transaction, staticAddress *resource.StaticAddress) error {
	var staticAddresses []*resource.StaticAddress
	if err := tx.Fill(map[string]interface{}{restdb.IDField: staticAddress.GetID()}, &staticAddresses); err != nil {
		return err
	}

	if len(staticAddresses) == 0 {
		return fmt.Errorf("no found static address %s", staticAddress.GetID())
	}

	staticAddress.Subnet = staticAddresses[0].Subnet
	staticAddress.Version = staticAddresses[0].Version
	staticAddress.HwAddress = staticAddresses[0].HwAddress
	staticAddress.IpAddress = staticAddresses[0].IpAddress
	staticAddress.Capacity = staticAddresses[0].Capacity
	return nil
}

func (s *StaticAddressHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	staticAddress := ctx.Resource.(*resource.StaticAddress)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := setStaticAddressFromDB(tx, staticAddress); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet, map[string]interface{}{
			"capacity": subnet.Capacity - staticAddress.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		if _, err := tx.Delete(resource.TableStaticAddress,
			map[string]interface{}{restdb.IDField: staticAddress.GetID()}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete static address %s with subnet %s failed: %s",
				staticAddress.String(), subnet.GetID(), err.Error()))
	}

	//eventbus.PublishResourceDeleteEvent(staticAddress)
	return nil
}