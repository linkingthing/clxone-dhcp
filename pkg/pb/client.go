package pb

import (
	"context"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/linkingthing/clxone-dhcp/config"
	"google.golang.org/grpc"
)

var connManager sync.Map

func NewConn(serviceName string) (*grpc.ClientConn, error) {
	if value, ok := connManager.Load(serviceName); ok {
		return value.(*grpc.ClientConn), nil
	}

	var logger kitlog.Logger
	{
		logger = kitlog.NewLogfmtLogger(os.Stderr)
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
		logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)
	}

	conf := consulapi.DefaultConfig()
	conf.Address = config.GetConfig().Consul.Address

	c, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}

	client := consul.NewClient(c)
	instance := consul.NewInstancer(client, logger, serviceName, []string{}, true)
	endpointor := sd.NewEndpointer(instance, getFactory, logger)

	balancer := lb.NewRoundRobin(endpointor)
	end, err := balancer.Endpoint()
	if err != nil {
		return nil, err
	}

	response, err := end(context.Background(), struct{}{})
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := grpc.DialContext(ctx,
		response.(string),
		grpc.WithBlock(),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, err
	}

	connManager.Store(serviceName, conn)

	return conn, nil
}

func GetEndpoints(serviceName string) ([]endpoint.Endpoint, error) {
	var logger kitlog.Logger
	{
		logger = kitlog.NewLogfmtLogger(os.Stderr)
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
		logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)
	}

	conf := consulapi.DefaultConfig()
	conf.Address = config.GetConfig().Consul.Address

	c, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}

	client := consul.NewClient(c)
	instance := consul.NewInstancer(client, logger, serviceName, []string{}, true)
	defer instance.Stop()

	endpointor := sd.NewEndpointer(instance, getFactory, logger)
	defer endpointor.Close()

	return endpointor.Endpoints()
}

func getFactory(instance string) (endpoint.Endpoint, io.Closer, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return instance, nil
	}, nil, nil
}
