package api

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"
)

const (
	ClientClassOption60 = "option[vendor-class-identifier].text == '%s'"
)

type ClientClassHandler struct {
}

func NewClientClassHandler() *ClientClassHandler {
	return &ClientClassHandler{}
}

func (c *ClientClassHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass)
	if clientclass.Name == "" || clientclass.Regexp == "" {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("clientclass params name %s and regexp %s must not be empty",
				clientclass.Name, clientclass.Regexp))
	}

	clientclass.SetID(clientclass.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientclass); err != nil {
			return err
		}

		return sendCreateClientClassCmdToAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add clientclass %s failed: %s", clientclass.Name, err.Error()))
	}

	return clientclass, nil
}

func sendCreateClientClassCmdToAgent(clientclass *resource.ClientClass) error {
	req, err := proto.Marshal(&dhcp_agent.CreateClientClass4Request{
		Name:   clientclass.Name,
		Regexp: fmt.Sprintf(ClientClassOption60, clientclass.Regexp),
	})

	if err != nil {
		return fmt.Errorf("marshal create clientclass request failed: %s", err.Error())
	}

	return services.NewDHCPAgentService().SendDHCPCmd(services.CreateClientClass4, req)
}

func (c *ClientClassHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var clientclasses []*resource.ClientClass
	if err := db.GetResources(map[string]interface{}{"orderby": "name"}, &clientclasses); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list clientclasses from db failed: %s", err.Error()))
	}

	return clientclasses, nil
}

func (c *ClientClassHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclassID := ctx.Resource.(*resource.ClientClass).GetID()
	var clientclasses []*resource.ClientClass
	clientclass, err := restdb.GetResourceWithID(db.GetDB(), clientclassID, &clientclasses)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get clientclass %s from db failed: %s", clientclassID, err.Error()))
	}

	return clientclass.(*resource.ClientClass), nil
}

func (c *ClientClassHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableClientClass, map[string]interface{}{
			"regexp": clientclass.Regexp,
		}, map[string]interface{}{restdb.IDField: clientclass.GetID()}); err != nil {
			return err
		}

		return sendUpdateClientClassCmdToDHCPAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update clientclass %s failed: %s", clientclass.GetID(), err.Error()))
	}

	return clientclass, nil
}

func sendUpdateClientClassCmdToDHCPAgent(clientclass *resource.ClientClass) error {
	req, err := proto.Marshal(&dhcp_agent.UpdateClientClass4Request{
		Name:   clientclass.Name,
		Regexp: fmt.Sprintf(ClientClassOption60, clientclass.Regexp),
	})

	if err != nil {
		return fmt.Errorf("marshal update clientclass request failed: %s", err.Error())
	}

	return services.NewDHCPAgentService().SendDHCPCmd(services.UpdateClientClass4, req)
}

func (c *ClientClassHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	clientclassID := ctx.Resource.(*resource.ClientClass).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableClientClass,
			map[string]interface{}{restdb.IDField: clientclassID}); err != nil {
			return err
		}

		return sendDeleteClientClassCmdToDHCPAgent(ctx.Resource.(*resource.ClientClass))
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete clientclass %s failed: %s", clientclassID, err.Error()))
	}

	return nil
}

func sendDeleteClientClassCmdToDHCPAgent(clientClass *resource.ClientClass) error {
	req, err := proto.Marshal(&dhcp_agent.DeleteClientClass4Request{
		Name: clientClass.GetID(),
	})

	if err != nil {
		return fmt.Errorf("marshal delete clientclass request failed: %s", err.Error())
	}

	return services.NewDHCPAgentService().SendDHCPCmd(services.DeleteClientClass4, req)
}
