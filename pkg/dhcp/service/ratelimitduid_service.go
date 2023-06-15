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

type RateLimitDuidService struct{}

func NewRateLimitDuidService() *RateLimitDuidService {
	return &RateLimitDuidService{}
}

func (d *RateLimitDuidService) Create(rateLimitDuid *resource.RateLimitDuid) error {
	if err := rateLimitDuid.Validate(); err != nil {
		return err
	}

	rateLimitDuid.SetID(rateLimitDuid.Duid)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rateLimitDuid); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameRateLimit), pg.Error(err).Error())
		}

		return sendCreateRateLimitDuidCmdToDHCPAgent(rateLimitDuid)
	})
}

func sendCreateRateLimitDuidCmdToDHCPAgent(rateLimitDuid *resource.RateLimitDuid) error {
	return kafka.SendDHCPCmd(kafka.CreateRateLimitDuid,
		&pbdhcpagent.CreateRateLimitDuidRequest{
			Duid:  rateLimitDuid.Duid,
			Limit: rateLimitDuid.RateLimit,
		}, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteRateLimitDuid,
				&pbdhcpagent.DeleteRateLimitDuidRequest{Duid: rateLimitDuid.Duid}); err != nil {
				log.Errorf("create ratelimit duid %s failed, rollback with nodes %v failed: %s",
					rateLimitDuid.Duid, nodesForSucceed, err.Error())
			}
		})
}

func (d *RateLimitDuidService) List(conditions map[string]interface{}) ([]*resource.RateLimitDuid, error) {
	var duids []*resource.RateLimitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &duids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameRateLimit), pg.Error(err).Error())
	}

	return duids, nil
}

func (d *RateLimitDuidService) Get(id string) (*resource.RateLimitDuid, error) {
	var rateLimitDuids []*resource.RateLimitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &rateLimitDuids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(rateLimitDuids) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
	}

	return rateLimitDuids[0], nil
}

func (d *RateLimitDuidService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitDuid, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
		}

		return sendDeleteRateLimitDuidCmdToDHCPAgent(id)
	})
}

func sendDeleteRateLimitDuidCmdToDHCPAgent(rateLimitDuidId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteRateLimitDuid,
		&pbdhcpagent.DeleteRateLimitDuidRequest{Duid: rateLimitDuidId}, nil)
}

func (d *RateLimitDuidService) Update(rateLimitDuid *resource.RateLimitDuid) error {
	if err := rateLimitDuid.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rateLimits []*resource.RateLimitDuid
		if err := tx.Fill(map[string]interface{}{restdb.IDField: rateLimitDuid.GetID()},
			&rateLimits); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, rateLimitDuid.GetID(), pg.Error(err).Error())
		} else if len(rateLimits) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, rateLimitDuid.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitDuid, map[string]interface{}{
			resource.SqlColumnRateLimit: rateLimitDuid.RateLimit,
			resource.SqlColumnComment:   rateLimitDuid.Comment,
		}, map[string]interface{}{restdb.IDField: rateLimitDuid.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, rateLimitDuid.GetID(), pg.Error(err).Error())
		}

		if rateLimits[0].RateLimit != rateLimitDuid.RateLimit {
			return sendUpdateRateLimitDuidCmdToDHCPAgent(rateLimitDuid)
		} else {
			return nil
		}
	})
}

func sendUpdateRateLimitDuidCmdToDHCPAgent(rateLimitDuid *resource.RateLimitDuid) error {
	return kafka.SendDHCPCmd(kafka.UpdateRateLimitDuid,
		&pbdhcpagent.UpdateRateLimitDuidRequest{
			Duid:  rateLimitDuid.Duid,
			Limit: rateLimitDuid.RateLimit,
		}, nil)
}
