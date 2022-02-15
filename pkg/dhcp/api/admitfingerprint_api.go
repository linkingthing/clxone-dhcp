package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type AdmitFingerprintApi struct {
	Service *service.AdmitFingerprintService
}

func NewAdmitFingerprintApi() *AdmitFingerprintApi {
	return &AdmitFingerprintApi{Service: service.NewAdmitFingerprintService()}
}

func (d *AdmitFingerprintApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprint := ctx.Resource.(*resource.AdmitFingerprint)
	admitFingerprint.SetID(admitFingerprint.ClientType)
	if err := admitFingerprint.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create admit fingerprint %s failed: %s", admitFingerprint.GetID(), err.Error()))
	}
	retAdmitFingerprint, err := d.Service.Create(admitFingerprint)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create admit fingerprint %s failed: %s", admitFingerprint.GetID(), err.Error()))
	}
	return retAdmitFingerprint, nil
}

func (d *AdmitFingerprintApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, err := d.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list admit fingerprints from db failed: %s", err.Error()))
	}
	return fingerprints, nil
}

func (d *AdmitFingerprintApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprintID := ctx.Resource.(*resource.AdmitFingerprint).GetID()
	admitFingerprint, err := d.Service.Get(admitFingerprintID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit fingerprint %s from db failed: %s", admitFingerprintID, err.Error()))
	}
	return admitFingerprint.(*resource.AdmitFingerprint), nil
}

func (d *AdmitFingerprintApi) Delete(ctx *restresource.Context) *resterror.APIError {
	admitFingerprintId := ctx.Resource.GetID()
	err := d.Service.Delete(admitFingerprintId)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete admit fingerprint %s failed: %s", admitFingerprintId, err.Error()))
	}
	return nil
}
