package pb

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
)

func NewClient(serviceName string) (*grpc.ClientConn, error) {
	var logger kitlog.Logger
	{
		logger = kitlog.NewLogfmtLogger(os.Stderr)
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
		logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)
	}

	conf := consulapi.DefaultConfig()
	conf.Address = "127.0.0.1:8500"

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
	conf.Address = "127.0.0.1:8500"

	c, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}

	client := consul.NewClient(c)
	instance := consul.NewInstancer(client, logger, serviceName, []string{}, true)
	endpointor := sd.NewEndpointer(instance, getFactory, logger)
	return endpointor.Endpoints()
}

func getFactory(instance string) (endpoint.Endpoint, io.Closer, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return instance, nil
	}, nil, nil
}
