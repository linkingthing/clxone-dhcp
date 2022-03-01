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

type PingerHandler struct {
}

func NewPingerHandler() (*PingerHandler, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TablePinger, nil); err != nil {
			return fmt.Errorf("check dhcp pinger failed: %s", err.Error())
		} else if exists == false {
			if _, err := tx.Insert(resource.DefaultPinger); err != nil {
				return fmt.Errorf("insert default dhcp pinger failed: %s", err.Error())
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &PingerHandler{}, nil
}

func (d *PingerHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var pingers []*resource.Pinger
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &pingers)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp pinger from db failed: %s", err.Error()))
	}

	return pingers, nil
}

func (d *PingerHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pingerID := ctx.Resource.GetID()
	var pingers []*resource.Pinger
	pinger, err := restdb.GetResourceWithID(db.GetDB(), pingerID, &pingers)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pinger from db failed: %s", err.Error()))
	}

	return pinger.(*resource.Pinger), nil
}

func (d *PingerHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pinger := ctx.Resource.(*resource.Pinger)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TablePinger, map[string]interface{}{
			"enabled": pinger.Enabled,
			"timeout": pinger.Timeout,
		}, map[string]interface{}{restdb.IDField: pinger.GetID()}); err != nil {
			return err
		}

		return sendUpdatePingerCmdToDHCPAgent(pinger)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp pinger failed: %s", err.Error()))
	}

	return pinger, nil
}

func sendUpdatePingerCmdToDHCPAgent(pinger *resource.Pinger) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdatePinger,
		&pbdhcpagent.UpdatePingerRequest{
			Enabled: pinger.Enabled,
			Timeout: pinger.Timeout,
		})
}
