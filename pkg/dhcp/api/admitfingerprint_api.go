package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type AdmitFingerprintApi struct {
	Service *service.AdmitFingerprintService
}

func NewAdmitFingerprintApi() *AdmitFingerprintApi {
	return &AdmitFingerprintApi{Service: service.NewAdmitFingerprintService()}
}

func (d *AdmitFingerprintApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprint := ctx.Resource.(*resource.AdmitFingerprint)
	if err := d.Service.Create(admitFingerprint); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitFingerprint, nil
}

func (d *AdmitFingerprintApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, err := d.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprints, nil
}

func (d *AdmitFingerprintApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprint, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitFingerprint, nil
}

func (d *AdmitFingerprintApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}
