package api

import (
	"fmt"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SharedNetwork4Handler struct{}

func NewSharedNetwork4Handler() *SharedNetwork4Handler {
	return &SharedNetwork4Handler{}
}

func (s *SharedNetwork4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	if err := sharedNetwork4.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create shared network4 params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(sharedNetwork4); err != nil {
			return err
		}

		return sendCreateSharedNetwork4CmdToDHCPAgent(sharedNetwork4)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create shared network4 %s failed: %s", sharedNetwork4.Name, err.Error()))
	}

	return sharedNetwork4, nil
}

func sendCreateSharedNetwork4CmdToDHCPAgent(sharedNetwork4 *resource.SharedNetwork4) error {
	err := dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateSharedNetwork4,
		sharedNetwork4ToCreateSharedNetwork4Request(sharedNetwork4))
	if err != nil {
		if err := sendDeleteSharedNetwork4CmdToDHCPAgent(sharedNetwork4.Name); err != nil {
			log.Errorf("create shared network4 %s failed, and rollback it failed: %s",
				sharedNetwork4.Name, err.Error())
		}
	}

	return err
}

func sharedNetwork4ToCreateSharedNetwork4Request(sharedNetwork4 *resource.SharedNetwork4) *dhcpagent.CreateSharedNetwork4Request {
	return &dhcpagent.CreateSharedNetwork4Request{
		Name:      sharedNetwork4.Name,
		SubnetIds: sharedNetwork4.SubnetIds,
	}
}

func (s *SharedNetwork4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var sharedNetwork4s []*resource.SharedNetwork4
	if err := db.GetResources(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		util.FilterNameName, util.FilterNameName), &sharedNetwork4s); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list shared network4s from db failed: %s", err.Error()))
	}

	return sharedNetwork4s, nil
}

func (s *SharedNetwork4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var err error
		sharedNetwork4, err = getOldSharedNetwork(tx, sharedNetwork4.GetID())
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get shared network4 %s failed: %s", sharedNetwork4.GetID(), err.Error()))
	}

	return sharedNetwork4, nil
}

func (s *SharedNetwork4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	if err := sharedNetwork4.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update shared network4 params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldSharedNetwork4, err := getOldSharedNetwork(tx, sharedNetwork4.GetID())
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSharedNetwork4, map[string]interface{}{
			"name":       sharedNetwork4.Name,
			"subnet_ids": sharedNetwork4.SubnetIds,
			"subnets":    sharedNetwork4.Subnets,
			"comment":    sharedNetwork4.Comment,
		}, map[string]interface{}{
			restdb.IDField: sharedNetwork4.GetID()}); err != nil {
			return err
		}

		return sendUpdateSharedNetwork4CmdToDHCPAgent(oldSharedNetwork4.Name, sharedNetwork4)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create shared network4 %s failed: %s", sharedNetwork4.Name, err.Error()))
	}

	return sharedNetwork4, nil
}

func sendUpdateSharedNetwork4CmdToDHCPAgent(name string, sharedNetwork4 *resource.SharedNetwork4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateSharedNetwork4,
		&dhcpagent.UpdateSharedNetwork4Request{
			Old: sharedNetworkNameToDeleteSharedNetwork4Request(name),
			New: sharedNetwork4ToCreateSharedNetwork4Request(sharedNetwork4),
		})
}

func (s *SharedNetwork4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	sharedNetwork4Id := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldSharedNetwork4, err := getOldSharedNetwork(tx, sharedNetwork4Id)
		if err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableSharedNetwork4, map[string]interface{}{
			restdb.IDField: sharedNetwork4Id}); err != nil {
			return err
		}

		return sendDeleteSharedNetwork4CmdToDHCPAgent(oldSharedNetwork4.Name)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete shared network4 %s failed: %s", sharedNetwork4Id, err.Error()))
	}

	return nil
}

func getOldSharedNetwork(tx restdb.Transaction, id string) (*resource.SharedNetwork4, error) {
	var sharedNetworks []*resource.SharedNetwork4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: id},
		&sharedNetworks); err != nil {
		return nil, err
	} else if len(sharedNetworks) == 0 {
		return nil, fmt.Errorf("no found shared network4")
	}

	return sharedNetworks[0], nil
}

func sendDeleteSharedNetwork4CmdToDHCPAgent(name string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteSharedNetwork4,
		sharedNetworkNameToDeleteSharedNetwork4Request(name))
}

func sharedNetworkNameToDeleteSharedNetwork4Request(name string) *dhcpagent.DeleteSharedNetwork4Request {
	return &dhcpagent.DeleteSharedNetwork4Request{Name: name}
}

func checkUsedBySharedNetwork(tx restdb.Transaction, subnetId uint64) error {
	var sharedNetwork4s []*resource.SharedNetwork4
	if err := tx.FillEx(&sharedNetwork4s,
		"select * from gr_shared_network4 where $1=any(subnet_ids)", subnetId); err != nil {
		return fmt.Errorf("check if it is used failed: %s", err.Error())
	} else if len(sharedNetwork4s) != 0 {
		return fmt.Errorf("used by shared network4 %s", sharedNetwork4s[0].Name)
	} else {
		return nil
	}
}
