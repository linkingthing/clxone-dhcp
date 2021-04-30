package client

import (
	"context"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/clients/user_service_client/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

func ValidateToken(token string, clientIP string) error {
	resolver.SetDefaultScheme("dns")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := grpc.DialContext(ctx,
		"dns:///consul.service.consul",
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithBalancerName("round_robin"))
	if err != nil {
		// log.Fatalf("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	c := pb.NewUserServiceClient(conn)

	{
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := c.CheckToken(ctx, &pb.CheckTokenRequest{
			Token:    token,
			ClientIP: clientIP,
		})
		return err

	}
}
