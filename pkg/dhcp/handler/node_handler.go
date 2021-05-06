package handler

import (
	"context"
	"io"
	"os"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/linkingthing/clxone-dhcp/config"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

type NodeHandler struct {
}

func NewNodeHandler(conf *config.DDIControllerConfig) *NodeHandler {
	return &NodeHandler{}
}

func (h *NodeHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {

	return nil, nil
}

func getNodeList() ([]endpoint.Endpoint, error) {
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	conf := consulapi.DefaultConfig()
	conf.Address = "127.0.0.1:8500"

	c, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}

	client := consul.NewClient(c)
	instance := consul.NewInstancer(client, logger, "clxone-user-grpc", []string{}, true)
	endpointor := sd.NewEndpointer(instance, getFactory, logger)
	return endpointor.Endpoints()
}

func getFactory(instance string) (endpoint.Endpoint, io.Closer, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return instance, nil
	}, nil, nil
}
