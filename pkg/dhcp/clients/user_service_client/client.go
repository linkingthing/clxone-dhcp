package client

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/clients/user_service_client/pb"
	"google.golang.org/grpc"
)

// func GetGrpcClient() (*grpc.ClientConn, error) {
// 	var conn *grpc.ClientConn
// 	resolver.SetDefaultScheme("dns")
// 	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
// 	conn, err := grpc.DialContext(ctx,
// 		"dns://127.0.0.1:8600/clxone-user-grpc.service.consul",
// 		grpc.WithBlock(),
// 		grpc.WithInsecure(),
// 		grpc.WithBalancerName("round_robin"))
// 	if err != nil {
// 		// log.Fatalf("did not connect: %v", err)
// 		return nil, err
// 	}
// 	return conn, err
// }

func NewClient() (*grpc.ClientConn, error) {
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
	endpointor := sd.NewEndpointer(instance, endFactory, logger)

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
		return nil, err
	}
	return conn, nil
}

func endFactory(instance string) (endpoint.Endpoint, io.Closer, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return instance, nil
	}, nil, nil
}

func ValidateToken(token string, clientIP string) error {
	// conn, err := GetGrpcClient()
	conn, err := NewClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	c := pb.NewUserServiceClient(conn)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = c.CheckToken(ctx, &pb.CheckTokenRequest{
		Token:    token,
		ClientIP: clientIP,
	})
	return err
}
