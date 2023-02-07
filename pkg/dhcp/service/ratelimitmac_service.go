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

type RateLimitMacService struct{}

func NewRateLimitMacService() *RateLimitMacService {
	return &RateLimitMacService{}
}

func (d *RateLimitMacService) Create(rateLimitMac *resource.RateLimitMac) error {
	if err := rateLimitMac.Validate(); err != nil {
		return err
	}

	rateLimitMac.SetID(rateLimitMac.HwAddress)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rateLimitMac); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameRateLimit), pg.Error(err).Error())
		}

		return sendCreateRateLimitMacCmdToDHCPAgent(rateLimitMac)
	})
}

func sendCreateRateLimitMacCmdToDHCPAgent(rateLimitMac *resource.RateLimitMac) error {
	return kafka.SendDHCPCmd(kafka.CreateRateLimitMac,
		&pbdhcpagent.CreateRateLimitMacRequest{
			HwAddress: rateLimitMac.HwAddress,
			Limit:     rateLimitMac.RateLimit,
		}, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteRateLimitMac, &pbdhcpagent.DeleteRateLimitMacRequest{
					HwAddress: rateLimitMac.HwAddress}); err != nil {
				log.Errorf("create ratelimit mac %s failed, rollback with nodes %v failed: %s",
					rateLimitMac.HwAddress, nodesForSucceed, err.Error())
			}
		})
}

func (d *RateLimitMacService) List(condition map[string]interface{}) ([]*resource.RateLimitMac, error) {
	var rateLimitMacs []*resource.RateLimitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(condition, &rateLimitMacs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameRateLimit), pg.Error(err).Error())
	}

	return rateLimitMacs, nil
}

func (d *RateLimitMacService) Get(id string) (*resource.RateLimitMac, error) {
	var rateLimitMacs []*resource.RateLimitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &rateLimitMacs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(rateLimitMacs) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
	}

	return rateLimitMacs[0], nil
}

func (d *RateLimitMacService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitMac, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
		}

		return sendDeleteRateLimitMacCmdToDHCPAgent(id)
	})
}

func sendDeleteRateLimitMacCmdToDHCPAgent(ratelimitMacId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteRateLimitMac,
		&pbdhcpagent.DeleteRateLimitMacRequest{HwAddress: ratelimitMacId}, nil)
}

func (d *RateLimitMacService) Update(rateLimitMac *resource.RateLimitMac) error {
	if err := rateLimitMac.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rateLimits []*resource.RateLimitMac
		if err := tx.Fill(map[string]interface{}{restdb.IDField: rateLimitMac.GetID()},
			&rateLimits); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, rateLimitMac.GetID(), pg.Error(err).Error())
		} else if len(rateLimits) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, rateLimitMac.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitMac, map[string]interface{}{
			resource.SqlColumnRateLimit: rateLimitMac.RateLimit,
			resource.SqlColumnComment:   rateLimitMac.Comment,
		}, map[string]interface{}{restdb.IDField: rateLimitMac.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, rateLimitMac.GetID(), pg.Error(err).Error())
		}

		if rateLimits[0].RateLimit != rateLimitMac.RateLimit {
			return sendUpdateRateLimitMacCmdToDHCPAgent(rateLimitMac)
		} else {
			return nil
		}
	})
}

func sendUpdateRateLimitMacCmdToDHCPAgent(ratelimitMac *resource.RateLimitMac) error {
	return kafka.SendDHCPCmd(kafka.UpdateRateLimitMac,
		&pbdhcpagent.UpdateRateLimitMacRequest{
			HwAddress: ratelimitMac.HwAddress,
			Limit:     ratelimitMac.RateLimit,
		}, nil)
}
