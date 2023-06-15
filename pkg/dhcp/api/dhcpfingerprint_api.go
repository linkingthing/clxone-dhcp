package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type DhcpFingerprintHandler struct {
	Service *service.DhcpFingerprintService
}

func NewDhcpFingerprintApi() *DhcpFingerprintHandler {
	return &DhcpFingerprintHandler{Service: service.NewDhcpFingerprintService()}
}

func (h *DhcpFingerprintHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.DhcpFingerprint)
	if err := h.Service.Create(fingerprint); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprint, nil
}

func (h *DhcpFingerprintHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, err := h.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		service.OrderByCreateTime, service.FingerprintFilterNames...))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprints, nil
}

func (h *DhcpFingerprintHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint, err := h.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprint, nil
}

func (h *DhcpFingerprintHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.DhcpFingerprint)
	if err := h.Service.Update(fingerprint); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprint, nil
}

func (h *DhcpFingerprintHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := h.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}
