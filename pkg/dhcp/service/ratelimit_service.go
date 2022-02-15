package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type RateLimitService struct {
}

func NewRateLimitService() *RateLimitService {
	return &RateLimitService{}
}

func (d *RateLimitService) CreateDefaultRateLimit() error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TableRateLimit, nil); err != nil {
			return fmt.Errorf("check dhcp ratelimit failed: %s", err.Error())
		} else if exists == false {
			if _, err := tx.Insert(resource.DefaultRateLimit); err != nil {
				return fmt.Errorf("insert default dhcp ratelimit failed: %s", err.Error())
			}
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (d *RateLimitService) List() (interface{}, error) {
	var ratelimits []*resource.RateLimit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &ratelimits)
	}); err != nil {
		return nil, fmt.Errorf("list dhcp ratelimit from db failed: %s", err.Error())
	}

	return ratelimits, nil
}

func (d *RateLimitService) Get(ratelimitID string) (restresource.Resource, error) {
	var ratelimits []*resource.RateLimit
	ratelimit, err := restdb.GetResourceWithID(db.GetDB(), ratelimitID, &ratelimits)
	if err != nil {
		return nil, err
	}

	return ratelimit.(*resource.RateLimit), nil
}

func (d *RateLimitService) Update(ratelimit *resource.RateLimit) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableRateLimit, map[string]interface{}{
			resource.SqlColumnEnabled: ratelimit.Enabled,
		}, map[string]interface{}{restdb.IDField: ratelimit.GetID()}); err != nil {
			return err
		}

		return sendUpdateRateLimitCmdToDHCPAgent(ratelimit)
	}); err != nil {
		return nil, err
	}

	return ratelimit, nil
}

func sendUpdateRateLimitCmdToDHCPAgent(ratelimit *resource.RateLimit) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateRateLimit,
		&pbdhcpagent.UpdateRateLimitRequest{
			Enabled: ratelimit.Enabled,
		})
}
