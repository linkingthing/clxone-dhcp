package service

import (
	"fmt"

	pg "github.com/cuityhj/gohelper/postgresql"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type RateLimitDuidService struct{}

func NewRateLimitDuidService() *RateLimitDuidService {
	return &RateLimitDuidService{}
}

func (d *RateLimitDuidService) Create(rateLimitDuid *resource.RateLimitDuid) error {
	if err := rateLimitDuid.Validate(); err != nil {
		return fmt.Errorf("validate ratelimit duid %s failed: %s", rateLimitDuid.Duid, err.Error())
	}

	rateLimitDuid.SetID(rateLimitDuid.Duid)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rateLimitDuid); err != nil {
			return pg.Error(err)
		}

		return sendCreateRateLimitDuidCmdToDHCPAgent(rateLimitDuid)
	}); err != nil {
		return fmt.Errorf("create ratelimit duid %s failed:%s", rateLimitDuid.Duid, err.Error())
	}

	return nil
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
		return nil, fmt.Errorf("list ratelimit duids from db failed: %s", pg.Error(err).Error())
	}

	return duids, nil
}

func (d *RateLimitDuidService) Get(id string) (*resource.RateLimitDuid, error) {
	var rateLimitDuids []*resource.RateLimitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &rateLimitDuids)
	}); err != nil {
		return nil, fmt.Errorf("get ratelimit duid %s failed:%s", id, pg.Error(err).Error())
	} else if len(rateLimitDuids) == 0 {
		return nil, fmt.Errorf("no found ratelimit duid %s", id)
	}

	return rateLimitDuids[0], nil
}

func (d *RateLimitDuidService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitDuid, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found ratelimit duid %s", id)
		}

		return sendDeleteRateLimitDuidCmdToDHCPAgent(id)
	}); err != nil {
		return fmt.Errorf("delete ratelimit duid %s failed:%s", id, err.Error())
	}

	return nil
}

func sendDeleteRateLimitDuidCmdToDHCPAgent(rateLimitDuidId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteRateLimitDuid,
		&pbdhcpagent.DeleteRateLimitDuidRequest{Duid: rateLimitDuidId}, nil)
}

func (d *RateLimitDuidService) Update(rateLimitDuid *resource.RateLimitDuid) error {
	if err := rateLimitDuid.Validate(); err != nil {
		return fmt.Errorf("validate ratelimit duid %s failed: %s", rateLimitDuid.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rateLimits []*resource.RateLimitDuid
		if err := tx.Fill(map[string]interface{}{restdb.IDField: rateLimitDuid.GetID()},
			&rateLimits); err != nil {
			return pg.Error(err)
		} else if len(rateLimits) == 0 {
			return fmt.Errorf("no found ratelimit duid %s", rateLimitDuid.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitDuid, map[string]interface{}{
			resource.SqlColumnRateLimit: rateLimitDuid.RateLimit,
			resource.SqlColumnComment:   rateLimitDuid.Comment,
		}, map[string]interface{}{restdb.IDField: rateLimitDuid.GetID()}); err != nil {
			return pg.Error(err)
		}

		if rateLimits[0].RateLimit != rateLimitDuid.RateLimit {
			return sendUpdateRateLimitDuidCmdToDHCPAgent(rateLimitDuid)
		} else {
			return nil
		}
	}); err != nil {
		return fmt.Errorf("update ratelimit duid %s failed:%s", rateLimitDuid.GetID(), err.Error())
	}

	return nil
}

func sendUpdateRateLimitDuidCmdToDHCPAgent(rateLimitDuid *resource.RateLimitDuid) error {
	return kafka.SendDHCPCmd(kafka.UpdateRateLimitDuid,
		&pbdhcpagent.UpdateRateLimitDuidRequest{
			Duid:  rateLimitDuid.Duid,
			Limit: rateLimitDuid.RateLimit,
		}, nil)
}
