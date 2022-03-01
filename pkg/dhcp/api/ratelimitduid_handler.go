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

type RateLimitDuidHandler struct{}

func NewRateLimitDuidHandler() *RateLimitDuidHandler {
	return &RateLimitDuidHandler{}
}

func (d *RateLimitDuidHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	ratelimitDuid.SetID(ratelimitDuid.Duid)
	if err := ratelimitDuid.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(ratelimitDuid); err != nil {
			return err
		}

		return sendCreateRateLimitDuidCmdToDHCPAgent(ratelimitDuid)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}

	return ratelimitDuid, nil
}

func sendCreateRateLimitDuidCmdToDHCPAgent(ratelimitDuid *resource.RateLimitDuid) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateRateLimitDuid,
		&pbdhcpagent.CreateRateLimitDuidRequest{
			Duid:  ratelimitDuid.Duid,
			Limit: ratelimitDuid.RateLimit,
		})
}

func (d *RateLimitDuidHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var duids []*resource.RateLimitDuid
	if err := db.GetResources(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		FieldDuid, FieldDuid), &duids); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list ratelimit duids from db failed: %s", err.Error()))
	}

	return duids, nil
}

func (d *RateLimitDuidHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitDuidID := ctx.Resource.GetID()
	var ratelimitDuids []*resource.RateLimitDuid
	ratelimitDuid, err := restdb.GetResourceWithID(db.GetDB(), ratelimitDuidID, &ratelimitDuids)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get ratelimit duid %s from db failed: %s", ratelimitDuidID, err.Error()))
	}

	return ratelimitDuid.(*resource.RateLimitDuid), nil
}

func (d *RateLimitDuidHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	ratelimitDuidId := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitDuid, map[string]interface{}{
			restdb.IDField: ratelimitDuidId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found ratelimit duid %s", ratelimitDuidId)
		}

		return sendDeleteRateLimitDuidCmdToDHCPAgent(ratelimitDuidId)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete ratelimit duid %s failed: %s", ratelimitDuidId, err.Error()))
	}

	return nil
}

func sendDeleteRateLimitDuidCmdToDHCPAgent(ratelimitDuidId string) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteRateLimitDuid,
		&pbdhcpagent.DeleteRateLimitDuidRequest{
			Duid: ratelimitDuidId,
		})
}

func (d *RateLimitDuidHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	if err := ratelimitDuid.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var ratelimits []*resource.RateLimitDuid
		if err := tx.Fill(map[string]interface{}{restdb.IDField: ratelimitDuid.GetID()},
			&ratelimits); err != nil {
			return err
		} else if len(ratelimits) == 0 {
			return fmt.Errorf("no found ratelimit duid %s", ratelimitDuid.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitDuid, map[string]interface{}{
			"rate_limit": ratelimitDuid.RateLimit,
			"comment":    ratelimitDuid.Comment,
		}, map[string]interface{}{restdb.IDField: ratelimitDuid.GetID()}); err != nil {
			return err
		}

		if ratelimits[0].RateLimit != ratelimitDuid.RateLimit {
			return sendUpdateRateLimitDuidCmdToDHCPAgent(ratelimitDuid)
		} else {
			return nil
		}
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}

	return ratelimitDuid, nil
}

func sendUpdateRateLimitDuidCmdToDHCPAgent(ratelimitDuid *resource.RateLimitDuid) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateRateLimitDuid,
		&pbdhcpagent.UpdateRateLimitDuidRequest{
			Duid:  ratelimitDuid.Duid,
			Limit: ratelimitDuid.RateLimit,
		})
}
