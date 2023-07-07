package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type DhcpConfigApi struct {
	Service *service.DhcpConfigService
}

func NewDhcpConfigApi() (*DhcpConfigApi, error) {
	s, err := service.NewDhcpConfigService()
	if err != nil {
		return nil, err
	}

	return &DhcpConfigApi{Service: s}, nil
}

func (d *DhcpConfigApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	configs, err := d.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return configs, nil
}

func (d *DhcpConfigApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	config, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return config, nil
}

func (d *DhcpConfigApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	config := ctx.Resource.(*resource.DhcpConfig)
	if err := d.Service.Update(config); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return config, nil
}
