package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type DhcpFingerprintHandler struct {
	Service *service.DhcpFingerprintService
}

func NewDhcpFingerprintApi() *DhcpFingerprintHandler {
	return &DhcpFingerprintHandler{Service: service.NewDhcpFingerprintService()}
}

func (h *DhcpFingerprintHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.DhcpFingerprint)
	if err := fingerprint.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("add fingerprint %s failed: %s",
				fingerprint.Fingerprint, err.Error()))
	}
	retFingerprint, err := h.Service.Create(fingerprint)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add fingerprint %s failed: %s",
				fingerprint.Fingerprint, err.Error()))
	}

	return retFingerprint, nil
}

func (h *DhcpFingerprintHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, err := h.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list fingerprints from db failed: %s", err.Error()))
	}

	return fingerprints, nil
}

func (h *DhcpFingerprintHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprintId := ctx.Resource.GetID()
	fingerprint, err := h.Service.Get(fingerprintId)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get fingerprint %s from db failed: %s",
				fingerprintId, err.Error()))
	}
	return fingerprint, nil
}

func (h *DhcpFingerprintHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.DhcpFingerprint)
	if err := fingerprint.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("add fingerprint %s failed: %s",
				fingerprint.Fingerprint, err.Error()))
	}
	retFingerprint, err := h.Service.Update(fingerprint)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update fingerprint %s failed: %s",
				fingerprint.GetID(), err.Error()))
	}
	return retFingerprint, nil
}

func (h *DhcpFingerprintHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	fingerprintId := ctx.Resource.GetID()
	err := h.Service.Delete(fingerprintId)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete fingerprint %s failed: %s", fingerprintId, err.Error()))
	}

	return nil
}
