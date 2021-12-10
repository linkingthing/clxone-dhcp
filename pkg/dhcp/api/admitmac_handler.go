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
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	FieldHwAddress = "hw_address"
)

type AdmitMacHandler struct{}

func NewAdmitMacHandler() *AdmitMacHandler {
	return &AdmitMacHandler{}
}

func (d *AdmitMacHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	admitMac.SetID(admitMac.HwAddress)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitMac); err != nil {
			return err
		}

		return sendCreateAdmitMacCmdToDHCPAgent(admitMac)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create admit mac %s failed: %s", admitMac.GetID(), err.Error()))
	}

	return admitMac, nil
}

func sendCreateAdmitMacCmdToDHCPAgent(admitMac *resource.AdmitMac) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateAdmitMac,
		&pbdhcpagent.CreateAdmitMacRequest{
			HwAddress: admitMac.HwAddress,
		})
}

func (d *AdmitMacHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var macs []*resource.AdmitMac
	if err := db.GetResources(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		FieldHwAddress, FieldHwAddress), &macs); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list admit macs from db failed: %s", err.Error()))
	}

	return macs, nil
}

func (d *AdmitMacHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMacID := ctx.Resource.(*resource.AdmitMac).GetID()
	var admitMacs []*resource.AdmitMac
	admitMac, err := restdb.GetResourceWithID(db.GetDB(), admitMacID, &admitMacs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit mac %s from db failed: %s", admitMacID, err.Error()))
	}

	return admitMac.(*resource.AdmitMac), nil
}

func (d *AdmitMacHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	admitMacId := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitMac, map[string]interface{}{
			restdb.IDField: admitMacId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit mac %s", admitMacId)
		}

		return sendDeleteAdmitMacCmdToDHCPAgent(admitMacId)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete admit mac %s failed: %s", admitMacId, err.Error()))
	}

	return nil
}

func sendDeleteAdmitMacCmdToDHCPAgent(admitMacId string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteAdmitMac,
		&pbdhcpagent.DeleteAdmitMacRequest{
			HwAddress: admitMacId,
		})
}

func (d *AdmitMacHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitMac, map[string]interface{}{
			"comment": admitMac.Comment,
		}, map[string]interface{}{restdb.IDField: admitMac.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit mac %s", admitMac.GetID())
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create admit mac %s failed: %s", admitMac.GetID(), err.Error()))
	}

	return admitMac, nil
}
