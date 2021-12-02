package pb

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"

	"github.com/linkingthing/clxone-dhcp/config"
)

var connManager sync.Map

func NewConn(serviceName string) (*grpc.ClientConn, error) {
	if value, ok := connManager.Load(serviceName); ok {
		return value.(*grpc.ClientConn), nil
	}

	endpointor, err := getDefaultEndpointer(serviceName)
	if err != nil {
		return nil, err
	}

	defer endpointor.Close()
	balancer := lb.NewRoundRobin(endpointor)
	roundRobinEndPoint, err := balancer.Endpoint()
	if err != nil {
		return nil, err
	}

	response, err := roundRobinEndPoint(context.Background(), struct{}{})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, response.(string), grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	connManager.Store(serviceName, conn)
	return conn, nil
}

func getDefaultEndpointer(serviceName string) (*sd.DefaultEndpointer, error) {
	conf := consulapi.DefaultConfig()
	conf.Address = config.GetConfig().Consul.Address
	apiClient, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}

	client := consul.NewClient(apiClient)
	logger := kitlog.With(kitlog.NewLogfmtLogger(os.Stderr), "timestamp", kitlog.DefaultTimestampUTC)
	instance := consul.NewInstancer(client, logger, serviceName, []string{}, true)
	defer instance.Stop()

	return sd.NewEndpointer(instance, getFactory, logger), nil
}

func CloseConns() {
	connManager.Range(func(k, v interface{}) bool {
		if conn, ok := v.(*grpc.ClientConn); ok {
			conn.Close()
		}
		return true
	})
}

func GetEndpoints(serviceName string) ([]endpoint.Endpoint, error) {
	endpointor, err := getDefaultEndpointer(serviceName)
	if err != nil {
		return nil, err
	}

	defer endpointor.Close()
	return endpointor.Endpoints()
}

func getFactory(instance string) (endpoint.Endpoint, io.Closer, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return instance, nil
	}, nil, nil
}
