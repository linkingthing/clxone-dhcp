package api

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type DhcpOuiHandler struct{}

func NewDhcpOuiHandler() *DhcpOuiHandler {
	return &DhcpOuiHandler{}
}

func (d *DhcpOuiHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpoui := ctx.Resource.(*resource.DhcpOui)
	if err := dhcpoui.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create dhcp oui %s failed: %s", dhcpoui.Oui, err.Error()))
	}

	dhcpoui.SetID(dhcpoui.Oui)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(dhcpoui); err != nil {
			return err
		}

		return sendCreateDhcpOuiCmdToDHCPAgent(dhcpoui)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create dhcp oui %s failed: %s", dhcpoui.GetID(), err.Error()))
	}

	return dhcpoui, nil
}

func sendCreateDhcpOuiCmdToDHCPAgent(dhcpoui *resource.DhcpOui) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateOui,
		&pbdhcpagent.CreateOuiRequest{
			Oui:          dhcpoui.Oui,
			Organization: dhcpoui.Organization,
		})
}

func (d *DhcpOuiHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var ouis []*resource.DhcpOui
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{"orderby": "oui"}, &ouis)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp ouis from db failed: %s", err.Error()))
	}

	return ouis, nil
}

func (d *DhcpOuiHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpouiID := ctx.Resource.(*resource.DhcpOui).GetID()
	var dhcpouis []*resource.DhcpOui
	dhcpoui, err := restdb.GetResourceWithID(db.GetDB(), dhcpouiID, &dhcpouis)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get dhcp oui %s from db failed: %s", dhcpouiID, err.Error()))
	}

	return dhcpoui.(*resource.DhcpOui), nil
}

func (d *DhcpOuiHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpoui := ctx.Resource.(*resource.DhcpOui)
	if err := dhcpoui.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update dhcp oui %s failed: %s", dhcpoui.Oui, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableDhcpOui, map[string]interface{}{
			"organization": dhcpoui.Organization,
		}, map[string]interface{}{restdb.IDField: dhcpoui.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found oui %s", dhcpoui.GetID())
		}

		return sendUpdateDhcpOuiCmdToDHCPAgent(dhcpoui)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp oui %s failed: %s", dhcpoui.GetID(), err.Error()))
	}

	return dhcpoui, nil
}

func sendUpdateDhcpOuiCmdToDHCPAgent(dhcpoui *resource.DhcpOui) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateOui,
		&pbdhcpagent.UpdateOuiRequest{
			Oui:          dhcpoui.Oui,
			Organization: dhcpoui.Organization,
		})
}

func (d *DhcpOuiHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	dhcpouiId := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableDhcpOui, map[string]interface{}{
			restdb.IDField: dhcpouiId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found oui %s", dhcpouiId)
		}

		return sendDeleteDhcpOuiCmdToDHCPAgent(dhcpouiId)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete dhcp oui %s failed: %s", dhcpouiId, err.Error()))
	}

	return nil
}

func sendDeleteDhcpOuiCmdToDHCPAgent(dhcpouiId string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteOui,
		&pbdhcpagent.DeleteOuiRequest{
			Oui: dhcpouiId,
		})
}
