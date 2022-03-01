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

type AdmitHandler struct {
}

func NewAdmitHandler() (*AdmitHandler, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TableAdmit, nil); err != nil {
			return fmt.Errorf("check dhcp admit failed: %s", err.Error())
		} else if exists == false {
			if _, err := tx.Insert(resource.DefaultAdmit); err != nil {
				return fmt.Errorf("insert default dhcp admit failed: %s", err.Error())
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &AdmitHandler{}, nil
}

func (d *AdmitHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var admits []*resource.Admit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &admits)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp admit from db failed: %s", err.Error()))
	}

	return admits, nil
}

func (d *AdmitHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitID := ctx.Resource.GetID()
	var admits []*resource.Admit
	admit, err := restdb.GetResourceWithID(db.GetDB(), admitID, &admits)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit from db failed: %s", err.Error()))
	}

	return admit.(*resource.Admit), nil
}

func (d *AdmitHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admit := ctx.Resource.(*resource.Admit)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableAdmit, map[string]interface{}{
			"enabled": admit.Enabled,
		}, map[string]interface{}{restdb.IDField: admit.GetID()}); err != nil {
			return err
		}

		return sendUpdateAdmitCmdToDHCPAgent(admit)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp admit failed: %s", err.Error()))
	}

	return admit, nil
}

func sendUpdateAdmitCmdToDHCPAgent(admit *resource.Admit) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateAdmit,
		&pbdhcpagent.UpdateAdmitRequest{
			Enabled: admit.Enabled,
		})
}
