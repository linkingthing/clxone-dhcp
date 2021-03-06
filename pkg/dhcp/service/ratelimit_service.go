package service

import (
	"fmt"

	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type RateLimitService struct {
}

func NewRateLimitService() (*RateLimitService, error) {
	if err := createDefaultRateLimit(); err != nil {
		return nil, err
	}

	return &RateLimitService{}, nil
}

func createDefaultRateLimit() error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TableRateLimit, nil); err != nil {
			return fmt.Errorf("check dhcp ratelimit failed: %s", pg.Error(err).Error())
		} else if !exists {
			if _, err := tx.Insert(resource.DefaultRateLimit); err != nil {
				return fmt.Errorf("insert default dhcp ratelimit failed: %s", pg.Error(err).Error())
			}
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (d *RateLimitService) List() ([]*resource.RateLimit, error) {
	var rateLimits []*resource.RateLimit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &rateLimits)
	}); err != nil {
		return nil, fmt.Errorf("list dhcp ratelimit failed: %s", pg.Error(err).Error())
	}

	return rateLimits, nil
}

func (d *RateLimitService) Get(id string) (*resource.RateLimit, error) {
	var rateLimits []*resource.RateLimit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &rateLimits)
	}); err != nil {
		return nil, fmt.Errorf("get ratelimit %s failed:%s", id, pg.Error(err).Error())
	} else if len(rateLimits) == 0 {
		return nil, fmt.Errorf("no found ratelimit %s", id)
	}

	return rateLimits[0], nil
}

func (d *RateLimitService) Update(rateLimit *resource.RateLimit) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableRateLimit, map[string]interface{}{
			resource.SqlColumnEnabled: rateLimit.Enabled,
		}, map[string]interface{}{restdb.IDField: rateLimit.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found ratelimit %s", rateLimit.GetID())
		}

		return sendUpdateRateLimitCmdToDHCPAgent(rateLimit)
	}); err != nil {
		return fmt.Errorf("update ratelimit %s failed:%s", rateLimit.GetID(), err.Error())
	}

	return nil
}

func sendUpdateRateLimitCmdToDHCPAgent(rateLimit *resource.RateLimit) error {
	return kafka.SendDHCPCmd(kafka.UpdateRateLimit,
		&pbdhcpagent.UpdateRateLimitRequest{Enabled: rateLimit.Enabled}, nil)
}
