package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type DhcpConfigApi struct {
	Service *service.DhcpConfigService
}

func NewDhcpConfigApi() *DhcpConfigApi {
	return &DhcpConfigApi{Service: service.NewDhcpConfigService()}
}

func (d *DhcpConfigApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	configs, err := d.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list global config from db failed: %s", err.Error()))
	}
	return configs, nil
}

func (d *DhcpConfigApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	configID := ctx.Resource.(*resource.DhcpConfig).GetID()
	config, err := d.Service.Get(configID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get global config %s from db failed: %s",
				configID, err.Error()))
	}
	return config, nil
}

func (d *DhcpConfigApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	config := ctx.Resource.(*resource.DhcpConfig)
	if err := config.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update global config params invalid: %s", err.Error()))
	}
	retConfig, err := d.Service.Update(config)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update global config %s failed: %s",
				config.GetID(), err.Error()))
	}
	return retConfig, nil
}
