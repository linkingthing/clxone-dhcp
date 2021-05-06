package handler

import (
	"context"
	"fmt"
	"io"

	"github.com/go-kit/kit/endpoint"
	"github.com/linkingthing/clxone-dhcp/config"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/dhcp/grpc_clients"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/sirupsen/logrus"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/sirupsen/logrus"

type NodeHandler struct {
}

func NewNodeHandler(conf *config.DDIControllerConfig) *NodeHandler {
	return &NodeHandler{}
}

func (h *NodeHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	getNodeList()

	return nil, nil
}

func getNodeList() (nodes []resource.Node, err error) {
	endpoints, err := grpcclient.GetEndpoints("clxone-dhcp-agent")
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("found clxone-dhcp-agent: %s", err.Error()))
	}
	for _, end := range endpoints {
		response, err := end(context.Background(), struct{}{})
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		nodes = append(nodes, resource.Node{
			Ip: response.(string),
		})
	}
	return
}

func getFactory(instance string) (endpoint.Endpoint, io.Closer, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return instance, nil
	}, nil, nil
}
