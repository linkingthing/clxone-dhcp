package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type RateLimitDuidService struct{}

func NewRateLimitDuidService() *RateLimitDuidService {
	return &RateLimitDuidService{}
}

func (d *RateLimitDuidService) Create(ratelimitDuid *resource.RateLimitDuid) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(ratelimitDuid); err != nil {
			return err
		}

		return sendCreateRateLimitDuidCmdToDHCPAgent(ratelimitDuid)
	}); err != nil {
		return nil, err
	}

	return ratelimitDuid, nil
}

func sendCreateRateLimitDuidCmdToDHCPAgent(ratelimitDuid *resource.RateLimitDuid) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateRateLimitDuid,
		&pbdhcpagent.CreateRateLimitDuidRequest{
			Duid:  ratelimitDuid.Duid,
			Limit: ratelimitDuid.RateLimit,
		})
}

func (d *RateLimitDuidService) List(ctx *restresource.Context) (interface{}, error) {
	var duids []*resource.RateLimitDuid
	if err := db.GetResources(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		FieldDuid, FieldDuid), &duids); err != nil {
		return nil, fmt.Errorf("list ratelimit duids from db failed: %s", err.Error())
	}

	return duids, nil
}

func (d *RateLimitDuidService) Get(ratelimitDuidID string) (restresource.Resource, error) {
	var ratelimitDuids []*resource.RateLimitDuid
	ratelimitDuid, err := restdb.GetResourceWithID(db.GetDB(), ratelimitDuidID, &ratelimitDuids)
	if err != nil {
		return nil, fmt.Errorf("get ratelimit duid %s from db failed: %s", ratelimitDuidID, err.Error())
	}

	return ratelimitDuid.(*resource.RateLimitDuid), nil
}

func (d *RateLimitDuidService) Delete(ratelimitDuidId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitDuid, map[string]interface{}{
			restdb.IDField: ratelimitDuidId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found ratelimit duid %s", ratelimitDuidId)
		}

		return sendDeleteRateLimitDuidCmdToDHCPAgent(ratelimitDuidId)
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteRateLimitDuidCmdToDHCPAgent(ratelimitDuidId string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteRateLimitDuid,
		&pbdhcpagent.DeleteRateLimitDuidRequest{
			Duid: ratelimitDuidId,
		})
}

func (d *RateLimitDuidService) Update(ratelimitDuid *resource.RateLimitDuid) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var ratelimits []*resource.RateLimitDuid
		if err := tx.Fill(map[string]interface{}{restdb.IDField: ratelimitDuid.GetID()},
			&ratelimits); err != nil {
			return err
		} else if len(ratelimits) == 0 {
			return fmt.Errorf("no found ratelimit duid %s", ratelimitDuid.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitDuid, map[string]interface{}{
			resource.SqlColumnRateLimit: ratelimitDuid.RateLimit,
			util.SqlColumnsComment:      ratelimitDuid.Comment,
		}, map[string]interface{}{restdb.IDField: ratelimitDuid.GetID()}); err != nil {
			return err
		}

		if ratelimits[0].RateLimit != ratelimitDuid.RateLimit {
			return sendUpdateRateLimitDuidCmdToDHCPAgent(ratelimitDuid)
		} else {
			return nil
		}
	}); err != nil {
		return nil, err
	}

	return ratelimitDuid, nil
}

func sendUpdateRateLimitDuidCmdToDHCPAgent(ratelimitDuid *resource.RateLimitDuid) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateRateLimitDuid,
		&pbdhcpagent.UpdateRateLimitDuidRequest{
			Duid:  ratelimitDuid.Duid,
			Limit: ratelimitDuid.RateLimit,
		})
}
