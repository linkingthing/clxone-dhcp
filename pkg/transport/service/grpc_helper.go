package service

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"

	consulutil "github.com/linkingthing/clxone-utils/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/linkingthing/clxone-dhcp/config"
)

var detectorNodeMap sync.Map

func getServiceConnect(hostname string, name string) (*grpc.ClientConn, error) {
	var target string
	if target_, ok := detectorNodeMap.Load(name); ok {
		target = target_.(string)
	} else {
		nodes, err := GetServiceNodes(name)
		if err != nil {
			return nil, err
		}
		if len(nodes) == 0 {
			return nil, fmt.Errorf("service %s has no nodes", name)
		}

		for _, node := range nodes {
			host, _, err := net.SplitHostPort(node)
			if err != nil {
				host = node
			}
			if host == hostname {
				target = node
				break
			}
		}
		if len(target) == 0 {
			return nil, fmt.Errorf("service %s with %s  has no target", name, hostname)
		}

		detectorNodeMap.Store(name, target)
	}

	return grpc.NewClient(fmt.Sprintf("passthrough:///%s", target),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)))
}

func GetServiceNodes(serviceName string) ([]string, error) {
	endPoints, err := consulutil.GetEndpoints(config.ConsulConfig, serviceName)
	if err != nil {
		return nil, err
	}

	nodes := make([]string, 0, len(endPoints))
	for _, endpoint := range endPoints {
		response, err := endpoint(context.Background(), struct{}{})
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, response.(string))
	}

	return nodes, nil
}
