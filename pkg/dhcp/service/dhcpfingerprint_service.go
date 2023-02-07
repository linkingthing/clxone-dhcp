package service

import (
	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
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

func (h *DhcpFingerprintService) Create(fingerprint *resource.DhcpFingerprint) error {
	if err := fingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(fingerprint); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameFingerprint), pg.Error(err).Error())
		}
		return sendCreateFingerprintCmdToAgent(fingerprint)
	})
}

func sendCreateFingerprintCmdToAgent(fingerprint *resource.DhcpFingerprint) error {
	return kafka.SendDHCPCmd(kafka.CreateFingerprint,
		fingerprintToCreateFingerprintRequest(fingerprint), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteFingerprint,
				fingerprintToDeleteFingerprintRequest(fingerprint)); err != nil {
				log.Errorf("create dhcp fingerprint %s failed, rollback with nodes %v failed: %s",
					fingerprint.Fingerprint, nodesForSucceed, err.Error())
			}
		})
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
	if len(vendorId) == 0 || matchPattern == resource.MatchPatternEqual {
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

func (h *DhcpFingerprintService) List(conditions map[string]interface{}) ([]*resource.DhcpFingerprint, error) {
	var fingerprints []*resource.DhcpFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &fingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameFingerprint), pg.Error(err).Error())
	}

	return fingerprints, nil
}

func (h *DhcpFingerprintService) Get(id string) (*resource.DhcpFingerprint, error) {
	var fingerprints []*resource.DhcpFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &fingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(fingerprints) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameFingerprint, id)
	}

	return fingerprints[0], nil
}

func (h *DhcpFingerprintService) Update(fingerprint *resource.DhcpFingerprint) error {
	if err := fingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldFingerprint, err := getFingerprintWithoutReadOnly(tx, fingerprint.GetID())
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableDhcpFingerprint, map[string]interface{}{
			resource.SqlColumnVendorId:        fingerprint.VendorId,
			resource.SqlColumnOperatingSystem: fingerprint.OperatingSystem,
			resource.SqlColumnClientType:      fingerprint.ClientType,
			resource.SqlColumnMatchPattern:    fingerprint.MatchPattern,
		}, map[string]interface{}{
			restdb.IDField: fingerprint.GetID(),
		}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, fingerprint.GetID(), pg.Error(err).Error())
		}

		return sendUpdateFingerprintCmdToDHCPAgent(oldFingerprint, fingerprint)
	})
}

func getFingerprintWithoutReadOnly(tx restdb.Transaction, id string) (*resource.DhcpFingerprint, error) {
	var fingerprints []*resource.DhcpFingerprint
	if err := tx.Fill(map[string]interface{}{restdb.IDField: id},
		&fingerprints); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(fingerprints) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameFingerprint, id)
	} else if fingerprints[0].IsReadOnly {
		return nil, errorno.ErrReadOnly(id)
	} else {
		return fingerprints[0], nil
	}
}

func sendUpdateFingerprintCmdToDHCPAgent(oldFingerprint, newFingerprint *resource.DhcpFingerprint) error {
	return kafka.SendDHCPCmd(kafka.UpdateFingerprint,
		&pbdhcpagent.UpdateFingerprintRequest{
			Old: fingerprintToDeleteFingerprintRequest(oldFingerprint),
			New: fingerprintToCreateFingerprintRequest(newFingerprint)}, nil)
}

func (h *DhcpFingerprintService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldFingerprint, err := getFingerprintWithoutReadOnly(tx, id)
		if err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableDhcpFingerprint, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		}

		return sendDeleteFingerprintCmdToDHCPAgent(oldFingerprint)
	})
}

func sendDeleteFingerprintCmdToDHCPAgent(oldFingerprint *resource.DhcpFingerprint) error {
	return kafka.SendDHCPCmd(kafka.DeleteFingerprint,
		fingerprintToDeleteFingerprintRequest(oldFingerprint), nil)
}

func fingerprintToDeleteFingerprintRequest(fingerprint *resource.DhcpFingerprint) *pbdhcpagent.DeleteFingerprintRequest {
	return &pbdhcpagent.DeleteFingerprintRequest{
		Fingerprint:     fingerprint.Fingerprint,
		VendorId:        getVendorIdByMatchPattern(fingerprint.VendorId, fingerprint.MatchPattern),
		OperatingSystem: fingerprint.OperatingSystem,
		ClientType:      fingerprint.ClientType,
	}
}
