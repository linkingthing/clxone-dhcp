package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zdnscloud/gorest"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/proto/logging"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
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

		auditLog := &logging.LoggingRequest{
			UserName:     "admin",
			SourceIp:     util.ClientIP(ctx.Request),
			Method:       method,
			ResourceKind: restresource.DefaultKindName(ctx.Resource),
			ResourcePath: ctx.Request.URL.Path,
			ResourceId:   ctx.Resource.GetID(),
			Parameters:   string(data),
			Success:      succeed,
			ErrMessage:   errMsg,
			Time:         time.Now().Format(time.RFC3339),
		}

		service.NewLoggingService().Log(auditLog)

		return nil
	}
}
