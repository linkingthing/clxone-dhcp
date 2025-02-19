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
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AdmitFingerprintService struct{}

func NewAdmitFingerprintService() *AdmitFingerprintService {
	return &AdmitFingerprintService{}
}

func (d *AdmitFingerprintService) Create(admitFingerprint *resource.AdmitFingerprint) error {
	admitFingerprint.SetID(admitFingerprint.ClientType)
	if err := admitFingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitFingerprint); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameFingerprint, admitFingerprint.ClientType, err)
		}

		return sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint)
	})
}

func sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint *resource.AdmitFingerprint) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitFingerprint,
		&pbdhcpagent.CreateAdmitFingerprintRequest{
			ClientType: admitFingerprint.ClientType,
			IsAdmitted: admitFingerprint.IsAdmitted,
		},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAdmitFingerprint,
				&pbdhcpagent.DeleteAdmitFingerprintRequest{
					ClientType: admitFingerprint.ClientType,
				}); err != nil {
				log.Errorf("create admit fingerprint %s failed, rollback with nodes %v failed: %s",
					admitFingerprint.ClientType, nodesForSucceed, err.Error())
			}
		})
}

func (d *AdmitFingerprintService) List() ([]*resource.AdmitFingerprint, error) {
	var fingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlOrderBy: resource.SqlColumnClientType}, &fingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAdmit), pg.Error(err).Error())
	}

	return fingerprints, nil
}

func (d *AdmitFingerprintService) Get(id string) (*resource.AdmitFingerprint, error) {
	var admitFingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitFingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(admitFingerprints) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAdmit, id)
	}

	return admitFingerprints[0], nil
}

func (d *AdmitFingerprintService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitFingerprint,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, id)
		}

		return sendDeleteAdmitFingerprintCmdToDHCPAgent(id)
	})
}

func sendDeleteAdmitFingerprintCmdToDHCPAgent(admitFingerprintId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitFingerprint,
		&pbdhcpagent.DeleteAdmitFingerprintRequest{ClientType: admitFingerprintId}, nil)
}

func (d *AdmitFingerprintService) Update(admitFingerprint *resource.AdmitFingerprint) error {
	if err := admitFingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var admitFingerprints []*resource.AdmitFingerprint
		if err := tx.Fill(map[string]interface{}{restdb.IDField: admitFingerprint.GetID()},
			&admitFingerprints); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, admitFingerprint.GetID(), pg.Error(err).Error())
		} else if len(admitFingerprints) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, admitFingerprint.GetID())
		}

		if _, err := tx.Update(resource.TableAdmitFingerprint,
			map[string]interface{}{
				resource.SqlColumnIsAdmitted: admitFingerprint.IsAdmitted,
				resource.SqlColumnComment:    admitFingerprint.Comment,
			},
			map[string]interface{}{restdb.IDField: admitFingerprint.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, admitFingerprint.GetID(), pg.Error(err).Error())
		}

		if admitFingerprints[0].IsAdmitted != admitFingerprint.IsAdmitted {
			return sendUpdateAdmitFingerprintCmdToDHCPAgent(admitFingerprint)
		} else {
			return nil
		}
	})
}

func sendUpdateAdmitFingerprintCmdToDHCPAgent(admitFingerprint *resource.AdmitFingerprint) error {
	return kafka.SendDHCPCmd(kafka.UpdateAdmitFingerprint,
		&pbdhcpagent.UpdateAdmitFingerprintRequest{
			ClientType: admitFingerprint.ClientType,
			IsAdmitted: admitFingerprint.IsAdmitted,
		}, nil)
}
