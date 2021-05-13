package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	"github.com/sirupsen/logrus"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

type NodeHandler struct{}

func NewNodeHandler() *NodeHandler {
	return &NodeHandler{}
}

func (h *NodeHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	nodes, err := services.NewDHCPService().GetNodeList()

	if err != nil {
		logrus.Error(err)
		return nil, resterror.NewAPIError(resterror.InvalidFormat, err.Error())
	}
	return nodes, nil
}
