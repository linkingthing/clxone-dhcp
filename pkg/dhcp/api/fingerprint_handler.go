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

type FingerprintHandler struct{}

func NewFingerprintHandler() *FingerprintHandler {
	return &FingerprintHandler{}
}

func (h *FingerprintHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.Fingerprint)
	if err := fingerprint.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("add fingerprint %s failed: %s", fingerprint.Fingerprint, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(fingerprint); err != nil {
			return err
		}

		return sendCreateFingerprintCmdToAgent(fingerprint)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add fingerprint %s failed: %s", fingerprint.Fingerprint, err.Error()))
	}

	return fingerprint, nil
}

func sendCreateFingerprintCmdToAgent(fingerprint *resource.Fingerprint) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateFingerprint,
		fingerprintToCreateFingerprintRequest(fingerprint))
}

func fingerprintToCreateFingerprintRequest(fingerprint *resource.Fingerprint) *dhcpagent.CreateFingerprintRequest {
	return &dhcpagent.CreateFingerprintRequest{
		Fingerprint:     fingerprint.Fingerprint,
		VendorId:        fingerprint.VendorId,
		OperatingSystem: fingerprint.OperatingSystem,
		ClientType:      fingerprint.ClientType,
		MatchPattern:    string(fingerprint.MatchPattern),
	}
}

func (h *FingerprintHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var fingerprints []*resource.Fingerprint
	if err := db.GetResources(map[string]interface{}{"orderby": restdb.IDField}, &fingerprints); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list fingerprints from db failed: %s", err.Error()))
	}

	return fingerprints, nil
}

func (h *FingerprintHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprintId := ctx.Resource.GetID()
	var fingerprints []*resource.Fingerprint
	_, err := restdb.GetResourceWithID(db.GetDB(), fingerprintId, &fingerprints)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get fingerprint %s from db failed: %s", fingerprintId, err.Error()))
	}

	return fingerprints[0], nil
}

func (h *FingerprintHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.Fingerprint)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var fingerprints []*resource.Fingerprint
		if err := tx.Fill(map[string]interface{}{restdb.IDField: fingerprint.GetID()},
			&fingerprints); err != nil {
			return err
		} else if len(fingerprints) == 0 {
			return fmt.Errorf("no found fingerprint %s", fingerprint.GetID())
		} else if fingerprints[0].IsReadOnly {
			return fmt.Errorf("update readonly fingerprint %s", fingerprint.GetID())
		}

		if _, err := tx.Update(resource.TableFingerprint, map[string]interface{}{
			"vendor_id":        fingerprint.VendorId,
			"operating_system": fingerprint.OperatingSystem,
			"client_type":      fingerprint.ClientType,
			"match_pattern":    fingerprint.MatchPattern,
		}, map[string]interface{}{
			restdb.IDField: fingerprint.GetID(),
		}); err != nil {
			return err
		}

		return sendUpdateFingerprintCmdToDHCPAgent(fingerprints[0], fingerprint)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update fingerprint %s failed: %s", fingerprint.GetID(), err.Error()))
	}

	return fingerprint, nil
}

func sendUpdateFingerprintCmdToDHCPAgent(oldFingerprint, newFingerprint *resource.Fingerprint) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateFingerprint,
		&dhcpagent.UpdateFingerprintRequest{
			Old: fingerprintToDeleteFingerprintRequest(oldFingerprint),
			New: fingerprintToCreateFingerprintRequest(newFingerprint)})
}

func (h *FingerprintHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	fingerprintId := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var fingerprints []*resource.Fingerprint
		if err := tx.Fill(map[string]interface{}{restdb.IDField: fingerprintId},
			&fingerprints); err != nil {
			return err
		} else if len(fingerprints) == 0 {
			return fmt.Errorf("no found fingerprint %s", fingerprintId)
		} else if fingerprints[0].IsReadOnly {
			return fmt.Errorf("update readonly fingerprint %s", fingerprintId)
		}

		if _, err := tx.Delete(resource.TableFingerprint, map[string]interface{}{
			restdb.IDField: fingerprintId}); err != nil {
			return err
		}

		return sendDeleteFingerprintCmdToDHCPAgent(fingerprints[0])
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete fingerprint %s failed: %s", fingerprintId, err.Error()))
	}

	return nil
}

func sendDeleteFingerprintCmdToDHCPAgent(oldFingerprint *resource.Fingerprint) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateFingerprint,
		fingerprintToDeleteFingerprintRequest(oldFingerprint))
}

func fingerprintToDeleteFingerprintRequest(fingerprint *resource.Fingerprint) *dhcpagent.DeleteFingerprintRequest {
	return &dhcpagent.DeleteFingerprintRequest{
		Fingerprint:     fingerprint.Fingerprint,
		VendorId:        fingerprint.VendorId,
		OperatingSystem: fingerprint.OperatingSystem,
		ClientType:      fingerprint.ClientType,
	}
}
