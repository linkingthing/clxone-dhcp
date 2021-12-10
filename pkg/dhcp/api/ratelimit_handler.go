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
)

type RateLimitHandler struct {
}

func NewRateLimitHandler() (*RateLimitHandler, error) {
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
		return nil, err
	}

	return &RateLimitHandler{}, nil
}

func (d *RateLimitHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var ratelimits []*resource.RateLimit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &ratelimits)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp ratelimit from db failed: %s", err.Error()))
	}

	return ratelimits, nil
}

func (d *RateLimitHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitID := ctx.Resource.(*resource.RateLimit).GetID()
	var ratelimits []*resource.RateLimit
	ratelimit, err := restdb.GetResourceWithID(db.GetDB(), ratelimitID, &ratelimits)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get ratelimit from db failed: %s", err.Error()))
	}

	return ratelimit.(*resource.RateLimit), nil
}

func (d *RateLimitHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimit := ctx.Resource.(*resource.RateLimit)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableRateLimit, map[string]interface{}{
			"enabled": ratelimit.Enabled,
		}, map[string]interface{}{restdb.IDField: ratelimit.GetID()}); err != nil {
			return err
		}

		return sendUpdateRateLimitCmdToDHCPAgent(ratelimit)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp ratelimit failed: %s", err.Error()))
	}

	return ratelimit, nil
}

func sendUpdateRateLimitCmdToDHCPAgent(ratelimit *resource.RateLimit) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateRateLimit,
		&pbdhcpagent.UpdateRateLimitRequest{
			Enabled: ratelimit.Enabled,
		})
}
