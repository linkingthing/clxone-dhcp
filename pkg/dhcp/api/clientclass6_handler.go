package api

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

const (
	ClientClass6Option60 = "option vendor-class-identifier == '%s'"
)

type ClientClass6Handler struct {
}

func NewClientClass6Handler() *ClientClass6Handler {
	return &ClientClass6Handler{}
}

func (c *ClientClass6Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass6)
	clientclass.SetID(clientclass.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientclass); err != nil {
			return err
		}

		return sendCreateClientClass6CmdToAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add clientclass %s failed: %s", clientclass.Name, err.Error()))
	}

	return clientclass, nil
}

func sendCreateClientClass6CmdToAgent(clientclass *resource.ClientClass6) error {
	err := dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateClientClass6,
		&pbdhcpagent.CreateClientClass6Request{
			Name:   clientclass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientclass.Regexp),
		})
	if err != nil {
		if err := sendDeleteClientClass6CmdToDHCPAgent(clientclass.Name); err != nil {
			log.Errorf("add clientclass6 %s failed, rollback it failed: %s",
				clientclass.Name, err.Error())
		}
	}

	return err
}

func (c *ClientClass6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var clientclasses []*resource.ClientClass6
	if err := db.GetResources(map[string]interface{}{"orderby": "name"},
		&clientclasses); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list clientclasses from db failed: %s", err.Error()))
	}

	return clientclasses, nil
}

func (c *ClientClass6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclassID := ctx.Resource.(*resource.ClientClass6).GetID()
	var clientclasses []*resource.ClientClass6
	clientclass, err := restdb.GetResourceWithID(db.GetDB(), clientclassID, &clientclasses)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get clientclass %s from db failed: %s",
				clientclassID, err.Error()))
	}

	return clientclass.(*resource.ClientClass6), nil
}

func (c *ClientClass6Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass6)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableClientClass6, map[string]interface{}{
			"regexp": clientclass.Regexp,
		}, map[string]interface{}{restdb.IDField: clientclass.GetID()}); err != nil {
			return err
		}

		return sendUpdateClientClass6CmdToDHCPAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update clientclass %s failed: %s",
				clientclass.GetID(), err.Error()))
	}

	return clientclass, nil
}

func sendUpdateClientClass6CmdToDHCPAgent(clientclass *resource.ClientClass6) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateClientClass6,
		&pbdhcpagent.UpdateClientClass6Request{
			Name:   clientclass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientclass.Regexp),
		})
}

func (c *ClientClass6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	clientclassID := ctx.Resource.(*resource.ClientClass6).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableSubnet6,
			map[string]interface{}{"client_class": clientclassID}); err != nil {
			return err
		} else if exist {
			return fmt.Errorf("client class %s used by subnet6", clientclassID)
		}

		if _, err := tx.Delete(resource.TableClientClass6,
			map[string]interface{}{restdb.IDField: clientclassID}); err != nil {
			return err
		}

		return sendDeleteClientClass6CmdToDHCPAgent(clientclassID)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete clientclass %s failed: %s", clientclassID, err.Error()))
	}

	return nil
}

func sendDeleteClientClass6CmdToDHCPAgent(clientClassID string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteClientClass6,
		&pbdhcpagent.DeleteClientClass6Request{
			Name: clientClassID,
		})
}
