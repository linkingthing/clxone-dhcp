package handler

import (
	"context"
	"fmt"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/pb"
	"github.com/sirupsen/logrus"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

type NodeHandler struct{}

func NewNodeHandler() *NodeHandler {
	return &NodeHandler{}
}

func (h *NodeHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	nodes, err := getDHCPNodeList()

	if err != nil {
		logrus.Error(err)
		return nil, resterror.NewAPIError(resterror.InvalidFormat, err.Error())
	}
	return nodes, nil
}

func getDHCPNodeList() (nodes []*resource.Node, err error) {
	endpoints, err := pb.GetEndpoints("clxone-dhcp-agent")
	if err != nil {
		logrus.Error(err)
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("found clxone-dhcp-agnet: %s", err.Error()))
	}
	for _, end := range endpoints {
		response, err := end(context.Background(), struct{}{})
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		logrus.Debug(response)

		// TODO : clxone-dhcp-agent should provider grpc method to get all nodes

		nodes = append(nodes, &resource.Node{})
	}

	return
}
