package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/util"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type AdmitFingerprintService struct{}

func NewAdmitFingerprintService() *AdmitFingerprintService {
	return &AdmitFingerprintService{}
}

func (d *AdmitFingerprintService) Create(admitFingerprint *resource.AdmitFingerprint) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitFingerprint); err != nil {
			return err
		}
		return sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint)
	}); err != nil {
		return nil, err
	}
	return admitFingerprint, nil
}

func sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint *resource.AdmitFingerprint) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateAdmitFingerprint,
		&pbdhcpagent.CreateAdmitFingerprintRequest{
			ClientType: admitFingerprint.ClientType,
		})
}

func (d *AdmitFingerprintService) List() (interface{}, error) {
	var fingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{util.SqlOrderBy: resource.AdmitFingerprintClientType}, &fingerprints)
	}); err != nil {
		return nil, err
	}
	return fingerprints, nil
}

func (d *AdmitFingerprintService) Get(admitFingerprintID string) (restresource.Resource, error) {
	var admitFingerprints []*resource.AdmitFingerprint
	admitFingerprint, err := restdb.GetResourceWithID(db.GetDB(), admitFingerprintID, &admitFingerprints)
	if err != nil {
		return nil, err
	}

	return admitFingerprint.(*resource.AdmitFingerprint), nil
}

func (d *AdmitFingerprintService) Delete(admitFingerprintId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitFingerprint,
			map[string]interface{}{restdb.IDField: admitFingerprintId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit fingerprint %s", admitFingerprintId)
		}
		return sendDeleteAdmitFingerprintCmdToDHCPAgent(admitFingerprintId)
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
