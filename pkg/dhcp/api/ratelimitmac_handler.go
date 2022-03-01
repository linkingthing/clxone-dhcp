package api

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type RateLimitMacHandler struct{}

func NewRateLimitMacHandler() *RateLimitMacHandler {
	return &RateLimitMacHandler{}
}

func (d *RateLimitMacHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitMac := ctx.Resource.(*resource.RateLimitMac)
	ratelimitMac.SetID(ratelimitMac.HwAddress)
	if err := ratelimitMac.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create ratelimit mac %s failed: %s", ratelimitMac.GetID(), err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(ratelimitMac); err != nil {
			return err
		}

		return sendCreateRateLimitMacCmdToDHCPAgent(ratelimitMac)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create ratelimit mac %s failed: %s", ratelimitMac.GetID(), err.Error()))
	}

	return ratelimitMac, nil
}

func sendCreateRateLimitMacCmdToDHCPAgent(ratelimitMac *resource.RateLimitMac) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateRateLimitMac,
		&pbdhcpagent.CreateRateLimitMacRequest{
			HwAddress: ratelimitMac.HwAddress,
			Limit:     ratelimitMac.RateLimit,
		})
}

func (d *RateLimitMacHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var macs []*resource.RateLimitMac
	if err := db.GetResources(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		FieldHwAddress, FieldHwAddress), &macs); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list ratelimit macs from db failed: %s", err.Error()))
	}

	return macs, nil
}

func (d *RateLimitMacHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitMacID := ctx.Resource.GetID()
	var ratelimitMacs []*resource.RateLimitMac
	ratelimitMac, err := restdb.GetResourceWithID(db.GetDB(), ratelimitMacID, &ratelimitMacs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get ratelimit mac %s from db failed: %s", ratelimitMacID, err.Error()))
	}

	return ratelimitMac.(*resource.RateLimitMac), nil
}

func (d *RateLimitMacHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	ratelimitMacId := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitMac, map[string]interface{}{
			restdb.IDField: ratelimitMacId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found ratelimit mac %s", ratelimitMacId)
		}

		return sendDeleteRateLimitMacCmdToDHCPAgent(ratelimitMacId)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete ratelimit mac %s failed: %s", ratelimitMacId, err.Error()))
	}

	return nil
}

func sendDeleteRateLimitMacCmdToDHCPAgent(ratelimitMacId string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteRateLimitMac,
		&pbdhcpagent.DeleteRateLimitMacRequest{
			HwAddress: ratelimitMacId,
		})
}

func (d *RateLimitMacHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitMac := ctx.Resource.(*resource.RateLimitMac)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var ratelimits []*resource.RateLimitMac
		if err := tx.Fill(map[string]interface{}{restdb.IDField: ratelimitMac.GetID()},
			&ratelimits); err != nil {
			return err
		} else if len(ratelimits) == 0 {
			return fmt.Errorf("no found ratelimit mac %s", ratelimitMac.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitMac, map[string]interface{}{
			"rate_limit": ratelimitMac.RateLimit,
			"comment":    ratelimitMac.Comment,
		}, map[string]interface{}{restdb.IDField: ratelimitMac.GetID()}); err != nil {
			return err
		}

		if ratelimits[0].RateLimit != ratelimitMac.RateLimit {
			return sendUpdateRateLimitMacCmdToDHCPAgent(ratelimitMac)
		} else {
			return nil
		}
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update ratelimit mac %s failed: %s", ratelimitMac.GetID(), err.Error()))
	}

	return ratelimitMac, nil
}

func sendUpdateRateLimitMacCmdToDHCPAgent(ratelimitMac *resource.RateLimitMac) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateRateLimitMac,
		&pbdhcpagent.UpdateRateLimitMacRequest{
			HwAddress: ratelimitMac.HwAddress,
			Limit:     ratelimitMac.RateLimit,
		})
}
