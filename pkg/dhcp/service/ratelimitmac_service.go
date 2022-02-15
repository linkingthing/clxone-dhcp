package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"

	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type RateLimitMacService struct{}

func NewRateLimitMacService() *RateLimitMacService {
	return &RateLimitMacService{}
}

func (d *RateLimitMacService) Create(ratelimitMac *resource.RateLimitMac) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(ratelimitMac); err != nil {
			return err
		}

		return sendCreateRateLimitMacCmdToDHCPAgent(ratelimitMac)
	}); err != nil {
		return nil, err
	}

	return ratelimitMac, nil
}

func sendCreateRateLimitMacCmdToDHCPAgent(ratelimitMac *resource.RateLimitMac) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateRateLimitMac,
		&pbdhcpagent.CreateRateLimitMacRequest{
			HwAddress: ratelimitMac.HwAddress,
			Limit:     ratelimitMac.RateLimit,
		})
}

func (d *RateLimitMacService) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var macs []*resource.RateLimitMac
	if err := db.GetResources(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnHwAddress, resource.SqlColumnHwAddress), &macs); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list ratelimit macs from db failed: %s", err.Error()))
	}

	return macs, nil
}

func (d *RateLimitMacService) Get(ratelimitMacID string) (restresource.Resource, error) {
	var ratelimitMacs []*resource.RateLimitMac
	ratelimitMac, err := restdb.GetResourceWithID(db.GetDB(), ratelimitMacID, &ratelimitMacs)
	if err != nil {
		return nil, err
	}

	return ratelimitMac.(*resource.RateLimitMac), nil
}

func (d *RateLimitMacService) Delete(ratelimitMacId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitMac, map[string]interface{}{
			restdb.IDField: ratelimitMacId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found ratelimit mac %s", ratelimitMacId)
		}

		return sendDeleteRateLimitMacCmdToDHCPAgent(ratelimitMacId)
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteRateLimitMacCmdToDHCPAgent(ratelimitMacId string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteRateLimitMac,
		&pbdhcpagent.DeleteRateLimitMacRequest{
			HwAddress: ratelimitMacId,
		})
}

func (d *RateLimitMacService) Update(ratelimitMac *resource.RateLimitMac) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var ratelimits []*resource.RateLimitMac
		if err := tx.Fill(map[string]interface{}{restdb.IDField: ratelimitMac.GetID()},
			&ratelimits); err != nil {
			return err
		} else if len(ratelimits) == 0 {
			return fmt.Errorf("no found ratelimit mac %s", ratelimitMac.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitMac, map[string]interface{}{
			resource.SqlColumnRateLimit: ratelimitMac.RateLimit,
			util.SqlColumnsComment:      ratelimitMac.Comment,
		}, map[string]interface{}{restdb.IDField: ratelimitMac.GetID()}); err != nil {
			return err
		}

		if ratelimits[0].RateLimit != ratelimitMac.RateLimit {
			return sendUpdateRateLimitMacCmdToDHCPAgent(ratelimitMac)
		} else {
			return nil
		}
	}); err != nil {
		return nil, err
	}

	return ratelimitMac, nil
}

func sendUpdateRateLimitMacCmdToDHCPAgent(ratelimitMac *resource.RateLimitMac) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateRateLimitMac,
		&pbdhcpagent.UpdateRateLimitMacRequest{
			HwAddress: ratelimitMac.HwAddress,
			Limit:     ratelimitMac.RateLimit,
		})
}
