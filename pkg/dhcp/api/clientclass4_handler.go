package api

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

const (
	ClientClass4Option60 = "option[vendor-class-identifier].text == '%s'"
)

type ClientClass4Handler struct {
}

func NewClientClass4Handler() *ClientClass4Handler {
	return &ClientClass4Handler{}
}

func (c *ClientClass4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass4)
	clientclass.SetID(clientclass.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientclass); err != nil {
			return err
		}

		return sendCreateClientClass4CmdToAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add clientclass %s failed: %s", clientclass.Name, err.Error()))
	}

	return clientclass, nil
}

func sendCreateClientClass4CmdToAgent(clientclass *resource.ClientClass4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateClientClass4,
		&dhcpagent.CreateClientClass4Request{
			Name:   clientclass.Name,
			Code:   clientclass.Code,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientclass.Regexp),
		})
}

func (c *ClientClass4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var clientclasses []*resource.ClientClass4
	if err := db.GetResources(map[string]interface{}{"orderby": "name"}, &clientclasses); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list clientclasses from db failed: %s", err.Error()))
	}

	return clientclasses, nil
}

func (c *ClientClass4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclassID := ctx.Resource.(*resource.ClientClass4).GetID()
	var clientclasses []*resource.ClientClass4
	clientclass, err := restdb.GetResourceWithID(db.GetDB(), clientclassID, &clientclasses)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get clientclass %s from db failed: %s", clientclassID, err.Error()))
	}

	return clientclass.(*resource.ClientClass4), nil
}

func (c *ClientClass4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableClientClass4, map[string]interface{}{
			"regexp": clientclass.Regexp,
		}, map[string]interface{}{restdb.IDField: clientclass.GetID()}); err != nil {
			return err
		}

		return sendUpdateClientClass4CmdToDHCPAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update clientclass %s failed: %s", clientclass.GetID(), err.Error()))
	}

	return clientclass, nil
}

func sendUpdateClientClass4CmdToDHCPAgent(clientclass *resource.ClientClass4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateClientClass4,
		&dhcpagent.UpdateClientClass4Request{
			Name:   clientclass.Name,
			Code:   clientclass.Code,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientclass.Regexp),
		})
}

func (c *ClientClass4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	clientclassID := ctx.Resource.(*resource.ClientClass4).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableClientClass4,
			map[string]interface{}{restdb.IDField: clientclassID}); err != nil {
			return err
		}

		return sendDeleteClientClass4CmdToDHCPAgent(clientclassID)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete clientclass %s failed: %s", clientclassID, err.Error()))
	}

	return nil
}

func sendDeleteClientClass4CmdToDHCPAgent(clientClassID string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteClientClass4,
		&dhcpagent.DeleteClientClass4Request{
			Name: clientClassID,
		})
}
