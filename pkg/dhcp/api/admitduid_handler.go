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
	FieldDuid = "duid"
)

type AdmitDuidHandler struct{}

func NewAdmitDuidHandler() *AdmitDuidHandler {
	return &AdmitDuidHandler{}
}

func (d *AdmitDuidHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid := ctx.Resource.(*resource.AdmitDuid)
	admitDuid.SetID(admitDuid.Duid)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitDuid); err != nil {
			return err
		}

		return sendCreateAdmitDuidCmdToDHCPAgent(admitDuid)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create admit duid %s failed: %s", admitDuid.GetID(), err.Error()))
	}

	return admitDuid, nil
}

func sendCreateAdmitDuidCmdToDHCPAgent(admitDuid *resource.AdmitDuid) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateAdmitDuid,
		&pbdhcpagent.CreateAdmitDuidRequest{
			Duid: admitDuid.Duid,
		})
}

func (d *AdmitDuidHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var duids []*resource.AdmitDuid
	if err := db.GetResources(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		FieldDuid, FieldDuid), &duids); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list admit duids from db failed: %s", err.Error()))
	}

	return duids, nil
}

func (d *AdmitDuidHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuidID := ctx.Resource.(*resource.AdmitDuid).GetID()
	var admitDuids []*resource.AdmitDuid
	admitDuid, err := restdb.GetResourceWithID(db.GetDB(), admitDuidID, &admitDuids)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit duid %s from db failed: %s", admitDuidID, err.Error()))
	}

	return admitDuid.(*resource.AdmitDuid), nil
}

func (d *AdmitDuidHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	admitDuidId := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitDuid, map[string]interface{}{
			restdb.IDField: admitDuidId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit duid %s", admitDuidId)
		}

		return sendDeleteAdmitDuidCmdToDHCPAgent(admitDuidId)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete admit duid %s failed: %s", admitDuidId, err.Error()))
	}

	return nil
}

func sendDeleteAdmitDuidCmdToDHCPAgent(admitDuidId string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteAdmitDuid,
		&pbdhcpagent.DeleteAdmitDuidRequest{
			Duid: admitDuidId,
		})
}

func (d *AdmitDuidHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid := ctx.Resource.(*resource.AdmitDuid)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitDuid, map[string]interface{}{
			"comment": admitDuid.Comment,
		}, map[string]interface{}{restdb.IDField: admitDuid.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit duid %s", admitDuid.GetID())
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update admit duid %s failed: %s", admitDuid.GetID(), err.Error()))
	}

	return admitDuid, nil
}
