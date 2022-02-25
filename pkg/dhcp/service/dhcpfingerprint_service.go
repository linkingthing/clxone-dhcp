package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var (
	Wildcard = []byte("%")
)

const (
	OrderByCreateTime = "create_time desc"
)

var FingerprintFilterNames = []string{"fingerprint", "vendor_id", "operating_system", "client_type"}

type DhcpFingerprintService struct{}

func NewDhcpFingerprintService() *DhcpFingerprintService {
	return &DhcpFingerprintService{}
}

func (h *DhcpFingerprintService) Create(fingerprint *resource.DhcpFingerprint) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(fingerprint); err != nil {
			return err
		}
		return sendCreateFingerprintCmdToAgent(fingerprint)
	}); err != nil {
		return nil, err
	}

	return fingerprint, nil
}

func sendCreateFingerprintCmdToAgent(fingerprint *resource.DhcpFingerprint) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateFingerprint,
		fingerprintToCreateFingerprintRequest(fingerprint))
}

func fingerprintToCreateFingerprintRequest(fingerprint *resource.DhcpFingerprint) *pbdhcpagent.CreateFingerprintRequest {
	return &pbdhcpagent.CreateFingerprintRequest{
		Fingerprint:     fingerprint.Fingerprint,
		VendorId:        getVendorIdByMatchPattern(fingerprint.VendorId, fingerprint.MatchPattern),
		OperatingSystem: fingerprint.OperatingSystem,
		ClientType:      fingerprint.ClientType,
		MatchPattern:    string(fingerprint.MatchPattern),
	}
}

func getVendorIdByMatchPattern(vendorId string, matchPattern resource.MatchPattern) string {
	if len(vendorId) == 0 {
		return vendorId
	}

	vendorBytes := []byte(vendorId)
	switch matchPattern {
	case resource.MatchPatternPrefix:
		vendorBytes = append(vendorBytes, Wildcard...)
	case resource.MatchPatternSuffix:
		vendorBytes = append(Wildcard, vendorBytes...)
	case resource.MatchPatternKeyword:
		vendorBytes = append(Wildcard, vendorBytes...)
		vendorBytes = append(vendorBytes, Wildcard...)
	}

	return string(vendorBytes)
}

func (h *DhcpFingerprintService) List(ctx *restresource.Context) (interface{}, error) {
	conditions := util.GenStrConditionsFromFilters(ctx.GetFilters(),
		OrderByCreateTime, FingerprintFilterNames...)
	var fingerprints []*resource.DhcpFingerprint
	if err := db.GetResources(conditions, &fingerprints); err != nil {
		return nil, err
	}
	return fingerprints, nil
}

func (h *DhcpFingerprintService) Get(fingerprintId string) (restresource.Resource, error) {
	var fingerprints []*resource.DhcpFingerprint
	_, err := restdb.GetResourceWithID(db.GetDB(), fingerprintId, &fingerprints)
	if err != nil {
		return nil, err
	}
	return fingerprints[0], nil
}

func (h *DhcpFingerprintService) Update(fingerprint *resource.DhcpFingerprint) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var fingerprints []*resource.DhcpFingerprint
		if err := tx.Fill(map[string]interface{}{restdb.IDField: fingerprint.GetID()},
			&fingerprints); err != nil {
			return err
		} else if len(fingerprints) == 0 {
			return fmt.Errorf("no found fingerprint %s", fingerprint.GetID())
		} else if fingerprints[0].IsReadOnly {
			return fmt.Errorf("update readonly fingerprint %s", fingerprint.GetID())
		}

		if _, err := tx.Update(resource.TableDhcpFingerprint, map[string]interface{}{
			resource.SqlDhcpFPrintVendorId:        fingerprint.VendorId,
			resource.SqlDhcpFPrintOperatingSystem: fingerprint.OperatingSystem,
			resource.SqlDhcpFPrintClientType:      fingerprint.ClientType,
			resource.SqlDhcpFPrintMatchPattern:    fingerprint.MatchPattern,
		}, map[string]interface{}{
			restdb.IDField: fingerprint.GetID(),
		}); err != nil {
			return err
		}

		return sendUpdateFingerprintCmdToDHCPAgent(fingerprints[0], fingerprint)
	}); err != nil {
		return nil, err
	}
	return fingerprint, nil
}

func sendUpdateFingerprintCmdToDHCPAgent(oldFingerprint, newFingerprint *resource.DhcpFingerprint) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateFingerprint,
		&pbdhcpagent.UpdateFingerprintRequest{
			Old: fingerprintToDeleteFingerprintRequest(oldFingerprint),
			New: fingerprintToCreateFingerprintRequest(newFingerprint)})
}

func (h *DhcpFingerprintService) Delete(fingerprintId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var fingerprints []*resource.DhcpFingerprint
		if err := tx.Fill(map[string]interface{}{restdb.IDField: fingerprintId},
			&fingerprints); err != nil {
			return err
		} else if len(fingerprints) == 0 {
			return fmt.Errorf("no found fingerprint %s", fingerprintId)
		} else if fingerprints[0].IsReadOnly {
			return fmt.Errorf("update readonly fingerprint %s", fingerprintId)
		}

		if _, err := tx.Delete(resource.TableDhcpFingerprint, map[string]interface{}{
			restdb.IDField: fingerprintId}); err != nil {
			return err
		}

		return sendDeleteFingerprintCmdToDHCPAgent(fingerprints[0])
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteFingerprintCmdToDHCPAgent(oldFingerprint *resource.DhcpFingerprint) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteFingerprint,
		fingerprintToDeleteFingerprintRequest(oldFingerprint))
}

func fingerprintToDeleteFingerprintRequest(fingerprint *resource.DhcpFingerprint) *pbdhcpagent.DeleteFingerprintRequest {
	return &pbdhcpagent.DeleteFingerprintRequest{
		Fingerprint:     fingerprint.Fingerprint,
		VendorId:        getVendorIdByMatchPattern(fingerprint.VendorId, fingerprint.MatchPattern),
		OperatingSystem: fingerprint.OperatingSystem,
		ClientType:      fingerprint.ClientType,
	}
}
