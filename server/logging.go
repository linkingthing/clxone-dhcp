package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/logging"
	"github.com/zdnscloud/gorest"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

var IgnoreAuditLog = "ignoreAuditLog"

func LoggingMiddleWare() gorest.EndHandlerFunc {
	return func(ctx *restresource.Context, respErr *resterror.APIError) *resterror.APIError {

		if _, ok := ctx.Get(IgnoreAuditLog); ok {
			return nil
		}

		var params interface{} = ctx.Resource
		method := ctx.Request.Method
		switch method {
		case http.MethodPost:
			if action := ctx.Resource.GetAction(); action != nil {
				method = action.Name
				params = action.Input
			}
		case http.MethodPut:
		case http.MethodDelete:
			params = nil
		default:
			return nil
		}

		data, err := json.Marshal(params)
		if err != nil {
			return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("marshal %s %s auditlog failed %s",
				method, ctx.Resource.GetType(), err.Error()))
		}

		var errMsg string
		succeed := true
		if respErr != nil {
			succeed = false
			errMsg = respErr.Error()
		}

		sourceIp := ctx.Request.RemoteAddr
		if strings.Contains(sourceIp, ":") {
			sourceIp = strings.Split(sourceIp, ":")[0]
		}

		auditLog := &logging.LoggingRequest{
			UserName:     "admin",
			SourceIp:     sourceIp,
			Method:       method,
			ResourceKind: restresource.DefaultKindName(ctx.Resource),
			ResourcePath: ctx.Request.URL.Path,
			ResourceId:   ctx.Resource.GetID(),
			Parameters:   string(data),
			Success:      succeed,
			ErrMessage:   errMsg,
			Time:         time.Now().Format(time.RFC3339),
		}

		services.NewLoggingService().Log(auditLog)

		return nil
	}
}