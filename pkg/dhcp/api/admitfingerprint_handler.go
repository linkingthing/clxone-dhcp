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

type AdmitFingerprintHandler struct{}

func NewAdmitFingerprintHandler() *AdmitFingerprintHandler {
	return &AdmitFingerprintHandler{}
}

func (d *AdmitFingerprintHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprint := ctx.Resource.(*resource.AdmitFingerprint)
	admitFingerprint.SetID(admitFingerprint.ClientType)
	if err := admitFingerprint.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create admit fingerprint %s failed: %s", admitFingerprint.GetID(), err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitFingerprint); err != nil {
			return err
		}

		return sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create admit fingerprint %s failed: %s", admitFingerprint.GetID(), err.Error()))
	}

	return admitFingerprint, nil
}

func sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint *resource.AdmitFingerprint) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateAdmitFingerprint,
		&pbdhcpagent.CreateAdmitFingerprintRequest{
			ClientType: admitFingerprint.ClientType,
		})
}

func (d *AdmitFingerprintHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var fingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{"orderby": "client_type"}, &fingerprints)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list admit fingerprints from db failed: %s", err.Error()))
	}

	return fingerprints, nil
}

func (d *AdmitFingerprintHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprintID := ctx.Resource.GetID()
	var admitFingerprints []*resource.AdmitFingerprint
	admitFingerprint, err := restdb.GetResourceWithID(db.GetDB(), admitFingerprintID, &admitFingerprints)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit fingerprint %s from db failed: %s", admitFingerprintID, err.Error()))
	}

	return admitFingerprint.(*resource.AdmitFingerprint), nil
}

func (d *AdmitFingerprintHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	admitFingerprintId := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitFingerprint, map[string]interface{}{
			restdb.IDField: admitFingerprintId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit fingerprint %s", admitFingerprintId)
		}

		return sendDeleteAdmitFingerprintCmdToDHCPAgent(admitFingerprintId)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete admit fingerprint %s failed: %s", admitFingerprintId, err.Error()))
	}

	return nil
}

func sendDeleteAdmitFingerprintCmdToDHCPAgent(admitFingerprintId string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteAdmitFingerprint,
		&pbdhcpagent.DeleteAdmitFingerprintRequest{
			ClientType: admitFingerprintId,
		})
}
