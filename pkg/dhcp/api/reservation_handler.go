package api

import (
	"context"
	"fmt"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"

	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type ReservationHandler struct {
}

func NewReservationHandler() *ReservationHandler {
	return &ReservationHandler{}
}

func (r *ReservationHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	reservation := ctx.Resource.(*resource.Reservation)
	if err := reservation.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create reservation params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkMacOrIpInUsed(tx, subnet.GetID(), reservation.HwAddress,
			reservation.IpAddress, false); err != nil {
			return err
		}

		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnetIfCanCreateDynamicPool(subnet); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet.Ipnet, reservation.IpAddress) == false {
			return fmt.Errorf("reservation ipaddress %s not belongs to subnet %s",
				reservation.IpAddress, subnet.Subnet)
		}

		if pdpool, conflict, err := checkIPConflictWithSubnetPDPool(tx,
			subnet.GetID(), reservation.IpAddress); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("reservation ipaddress %s conflicts with pdpool %s in subnet %s",
				reservation.IpAddress, pdpool, subnet.GetID())
		}

		conflictPool, conflict, err := checkPoolConflictWithSubnetPool(tx, subnet.GetID(),
			&resource.Pool{BeginAddress: reservation.IpAddress, EndAddress: reservation.IpAddress})
		if err != nil {
			return err
		}

		if conflict == false {
			if _, err := tx.Update(resource.TableSubnet, map[string]interface{}{
				"capacity": subnet.Capacity + 1,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TablePool, map[string]interface{}{
				"capacity": conflictPool.Capacity - 1,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s", conflictPool.String(), err.Error())
			}
		}

		reservation.Capacity = 1
		reservation.Subnet = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return err
		}

		return sendCreateReservationCmdToDDIAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create reservation with mac %s failed: %s", reservation.HwAddress, err.Error()))
	}

	//eventbus.PublishResourceCreateEvent(reservation)
	return reservation, nil
}

func sendCreateReservationCmdToDDIAgent(subnetID uint32, reservation *resource.Reservation) error {
	var req []byte
	var err error
	cmd := services.CreateReservation4
	if reservation.Version == util.IPVersion4 {
		req, err = proto.Marshal(&dhcp_agent.CreateReservation4Request{
			SubnetId:      subnetID,
			HwAddress:     reservation.HwAddress,
			IpAddress:     reservation.IpAddress,
			DomainServers: reservation.DomainServers,
			Routers:       reservation.Routers,
		})
	} else {
		cmd = services.CreateReservation6
		req, err = proto.Marshal(&dhcp_agent.CreateReservation6Request{
			SubnetId:    subnetID,
			HwAddress:   reservation.HwAddress,
			IpAddresses: []string{reservation.IpAddress},
			DnsServers:  reservation.DomainServers,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal create reservation request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
	return services.NewDHCPAgentService().SendDHCPCmd(cmd, req)
}

func (r *ReservationHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var reservations resource.Reservations
	if err := db.GetResources(map[string]interface{}{"subnet": subnetID}, &reservations); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reservations with subnet %s from db failed: %s", subnetID, err.Error()))
	}

	for _, reservation := range reservations {
		if err := setReservationLeasesUsedRatio(reservation); err != nil {
			log.Warnf("get reservation %s with subnet %s leases used ratio failed: %s",
				reservation.String(), subnetID, err.Error())
		}
	}

	sort.Sort(reservations)
	return reservations, nil
}

func (r *ReservationHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservationID := ctx.Resource.GetID()
	var reservations []*resource.Reservation
	reservationInterface, err := restdb.GetResourceWithID(db.GetDB(), reservationID, &reservations)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get reservation %s with subnetID %s from db failed: %s",
				reservationID, subnetID, err.Error()))
	}

	reservation := reservationInterface.(*resource.Reservation)
	if err := setReservationLeasesUsedRatio(reservation); err != nil {
		log.Warnf("get reservation %s with subnet %s leases used ratio failed: %s",
			reservation.String(), subnetID, err.Error())
	}
	return reservation, nil
}

func setReservationLeasesUsedRatio(reservation *resource.Reservation) error {
	leasesCount, err := getReservationLeasesCount(reservation)
	if err != nil {
		return err
	}

	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(reservation.Capacity))
	}
	return nil
}

func getReservationLeasesCount(reservation *resource.Reservation) (uint64, error) {
	if reservation.Capacity == 0 {
		return 0, nil
	}

	var resp *dhcp_agent.GetLeasesCountResponse
	var err error
	if reservation.Version == util.IPVersion4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetReservation4LeasesCount(context.TODO(),
			&dhcp_agent.GetReservation4LeasesCountRequest{
				SubnetId:  subnetIDStrToUint32(reservation.Subnet),
				HwAddress: reservation.HwAddress,
			})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetReservation6LeasesCount(context.TODO(),
			&dhcp_agent.GetReservation6LeasesCountRequest{
				SubnetId:  subnetIDStrToUint32(reservation.Subnet),
				HwAddress: reservation.HwAddress,
			})
	}

	return resp.GetLeasesCount(), err
}

func (r *ReservationHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservation := ctx.Resource.(*resource.Reservation)
	if err := reservation.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update reservation params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setReservationFromDB(tx, reservation); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableReservation, map[string]interface{}{
			"domain_servers": reservation.DomainServers,
			"routers":        reservation.Routers,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendUpdateReservationCmdToDDIAgent(subnetID, reservation)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update reservation %s with subnet %s failed: %s",
				reservation.GetID(), subnetID, err.Error()))
	}

	return reservation, nil
}

func setReservationFromDB(tx restdb.Transaction, reservation *resource.Reservation) error {
	var reservations []*resource.Reservation
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()}, &reservations); err != nil {
		return err
	}

	if len(reservations) == 0 {
		return fmt.Errorf("no found reservation %s", reservation.GetID())
	}

	reservation.Subnet = reservations[0].Subnet
	reservation.Version = reservations[0].Version
	reservation.HwAddress = reservations[0].HwAddress
	reservation.IpAddress = reservations[0].IpAddress
	reservation.Capacity = reservations[0].Capacity
	return nil
}

func sendUpdateReservationCmdToDDIAgent(subnetID string, reservation *resource.Reservation) error {
	var req []byte
	var err error
	cmd := services.UpdateReservation4
	if reservation.Version == util.IPVersion4 {
		req, err = proto.Marshal(&dhcp_agent.UpdateReservation4Request{
			SubnetId:      subnetIDStrToUint32(subnetID),
			HwAddress:     reservation.HwAddress,
			DomainServers: reservation.DomainServers,
			Routers:       reservation.Routers,
		})
	} else {
		cmd = services.UpdateReservation6
		req, err = proto.Marshal(&dhcp_agent.UpdateReservation6Request{
			SubnetId:   subnetIDStrToUint32(subnetID),
			HwAddress:  reservation.HwAddress,
			DnsServers: reservation.DomainServers,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal update reservation request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
	return services.NewDHCPAgentService().SendDHCPCmd(cmd, req)
}

func (r *ReservationHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	reservation := ctx.Resource.(*resource.Reservation)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservationFromDB(tx, reservation); err != nil {
			return err
		}

		if leasesCount, err := getReservationLeasesCount(reservation); err != nil {
			return fmt.Errorf("get reservation %s leases count failed: %s", reservation.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete reservation with %d ips had been allocated", leasesCount)
		}

		conflictPool, conflict, err := checkPoolConflictWithSubnetPool(tx, subnet.GetID(),
			&resource.Pool{BeginAddress: reservation.IpAddress, EndAddress: reservation.IpAddress})
		if err != nil {
			return err
		} else if conflict {
			if _, err := tx.Update(resource.TablePool, map[string]interface{}{
				"capacity": conflictPool.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s", conflictPool.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TableSubnet, map[string]interface{}{
				"capacity": subnet.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
			}
		}

		if _, err := tx.Delete(resource.TableReservation,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservationCmdToDDIAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete reservation %s with subnet %s failed: %s",
				reservation.String(), subnet.GetID(), err.Error()))
	}

	//eventbus.PublishResourceDeleteEvent(reservation)
	return nil
}

func sendDeleteReservationCmdToDDIAgent(subnetID uint32, reservation *resource.Reservation) error {
	var req []byte
	var err error
	cmd := services.DeleteReservation4
	if reservation.Version == util.IPVersion4 {
		req, err = proto.Marshal(&dhcp_agent.DeleteReservation4Request{
			SubnetId:  subnetID,
			HwAddress: reservation.HwAddress,
		})
	} else {
		cmd = services.DeleteReservation6
		req, err = proto.Marshal(&dhcp_agent.DeleteReservation6Request{
			SubnetId:  subnetID,
			HwAddress: reservation.HwAddress,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal delete reservation request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
	return services.NewDHCPAgentService().SendDHCPCmd(cmd, req)
}
