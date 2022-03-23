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

type RateLimitMacService struct{}

func NewRateLimitMacService() *RateLimitMacService {
	return &RateLimitMacService{}
}

func (d *RateLimitMacService) Create(rateLimitMac *resource.RateLimitMac) error {
	if err := rateLimitMac.Validate(); err != nil {
		return fmt.Errorf("validate ratelimit mac %s failed: %s", rateLimitMac.GetID(), err.Error())
	}

	rateLimitMac.SetID(rateLimitMac.HwAddress)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rateLimitMac); err != nil {
			return err
		}

		return sendCreateRateLimitMacCmdToDHCPAgent(rateLimitMac)
	}); err != nil {
		return fmt.Errorf("create ratelimit mac %s failed:%s", rateLimitMac.GetID(), err.Error())
	}

	return nil
}

func sendCreateRateLimitMacCmdToDHCPAgent(rateLimitMac *resource.RateLimitMac) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateRateLimitMac,
		&pbdhcpagent.CreateRateLimitMacRequest{
			HwAddress: rateLimitMac.HwAddress,
			Limit:     rateLimitMac.RateLimit,
		})
}

func (d *RateLimitMacService) List(condition map[string]interface{}) ([]*resource.RateLimitMac, error) {
	var rateLimitMacs []*resource.RateLimitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(condition, &rateLimitMacs)
	}); err != nil {
		return nil, fmt.Errorf("list ratelimit macs from db failed: %s", err.Error())
	}

	return rateLimitMacs, nil
}

func (d *RateLimitMacService) Get(id string) (*resource.RateLimitMac, error) {
	var rateLimitMacs []*resource.RateLimitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &rateLimitMacs)
	}); err != nil {
		return nil, fmt.Errorf("get ratelimit mac %s failed:%s", id, err.Error())
	} else if len(rateLimitMacs) == 0 {
		return nil, fmt.Errorf("no found ratelimit mac %s", id)
	}

	return rateLimitMacs[0], nil
}

func (d *RateLimitMacService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitMac, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found ratelimit mac %s", id)
		}

		return sendDeleteRateLimitMacCmdToDHCPAgent(id)
	}); err != nil {
		return fmt.Errorf("delete ratelimit mac %s failed:%s", id, err.Error())
	}

	return nil
}

func sendDeleteRateLimitMacCmdToDHCPAgent(ratelimitMacId string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteRateLimitMac,
		&pbdhcpagent.DeleteRateLimitMacRequest{
			HwAddress: ratelimitMacId,
		})
}

func (d *RateLimitMacService) Update(rateLimitMac *resource.RateLimitMac) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rateLimits []*resource.RateLimitMac
		if err := tx.Fill(map[string]interface{}{restdb.IDField: rateLimitMac.GetID()},
			&rateLimits); err != nil {
			return err
		} else if len(rateLimits) == 0 {
			return fmt.Errorf("no found ratelimit mac %s", rateLimitMac.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitMac, map[string]interface{}{
			resource.SqlColumnRateLimit: rateLimitMac.RateLimit,
			util.SqlColumnsComment:      rateLimitMac.Comment,
		}, map[string]interface{}{restdb.IDField: rateLimitMac.GetID()}); err != nil {
			return err
		}

		if rateLimits[0].RateLimit != rateLimitMac.RateLimit {
			return sendUpdateRateLimitMacCmdToDHCPAgent(rateLimitMac)
		} else {
			return nil
		}
	}); err != nil {
		return fmt.Errorf("update ratelimit mac %s failed:%s", rateLimitMac.GetID(), err.Error())
	}

	return nil
}

func sendUpdateRateLimitMacCmdToDHCPAgent(ratelimitMac *resource.RateLimitMac) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateRateLimitMac,
		&pbdhcpagent.UpdateRateLimitMacRequest{
			HwAddress: ratelimitMac.HwAddress,
			Limit:     ratelimitMac.RateLimit,
		})
}
