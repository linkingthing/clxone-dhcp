package handler

import (
	"net"

	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	FilterNameInput = "input"
)

type AssetSearchHandler struct{}

func NewAssetSearchHandler() *AssetSearchHandler {
	return &AssetSearchHandler{}
}

func (h *AssetSearchHandler) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	input, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNameInput, ctx.GetFilters())
	if ok == false || util.IsSpaceField(input) {
		return nil, nil
	}

	if ip := net.ParseIP(input); ip != nil {
		return getIpAssetInfo(ctx, ip)
	} else {
		return getDomainAssetInfo(ctx, input)
	}
}
