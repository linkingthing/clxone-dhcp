package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AdmitFingerprintService struct{}

func NewAdmitFingerprintService() *AdmitFingerprintService {
	return &AdmitFingerprintService{}
}

func (d *AdmitFingerprintService) Create(admitFingerprint *resource.AdmitFingerprint) error {
	admitFingerprint.SetID(admitFingerprint.ClientType)
	if err := admitFingerprint.Validate(); err != nil {
		return fmt.Errorf("create admit fingerprint %s failed: %s",
			admitFingerprint.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitFingerprint); err != nil {
			return err
		}

		return sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint)
	}); err != nil {
		return fmt.Errorf("create admit fingerprint failed:%s", err.Error())
	}

	return nil
}

func sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint *resource.AdmitFingerprint) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateAdmitFingerprint,
		&pbdhcpagent.CreateAdmitFingerprintRequest{
			ClientType: admitFingerprint.ClientType,
		})
}

func (d *AdmitFingerprintService) List() ([]*resource.AdmitFingerprint, error) {
	var fingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			util.SqlOrderBy: resource.AdmitFingerprintClientType}, &fingerprints)
	}); err != nil {
		return nil, fmt.Errorf("list admit fingerprints from db failed: %s", err.Error())
	}

	return fingerprints, nil
}

func (d *AdmitFingerprintService) Get(id string) (*resource.AdmitFingerprint, error) {
	var admitFingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitFingerprints)
	}); err != nil {
		return nil, fmt.Errorf("get admit fingerprint %s failed:%s", id, err.Error())
	} else if len(admitFingerprints) == 0 {
		return nil, fmt.Errorf("get admit fingerprint of %s failed: record not found", id)
	}

	return admitFingerprints[0], nil
}

func (d *AdmitFingerprintService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitFingerprint,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit fingerprint %s", id)
		}

		return sendDeleteAdmitFingerprintCmdToDHCPAgent(id)
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteAdmitFingerprintCmdToDHCPAgent(admitFingerprintId string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteAdmitFingerprint,
		&pbdhcpagent.DeleteAdmitFingerprintRequest{
			ClientType: admitFingerprintId,
		})
}
