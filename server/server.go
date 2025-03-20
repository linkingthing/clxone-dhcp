package server

import (
	"fmt"
	"google.golang.org/grpc/keepalive"
	"net"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linkingthing/clxone-utils/excel"
	"github.com/linkingthing/gorest"
	"github.com/linkingthing/gorest/adaptor"
	"github.com/linkingthing/gorest/resource/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	hv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp"
	"github.com/linkingthing/clxone-dhcp/pkg/grpc/service"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

const (
	HealthPath = "/health"
)

var LoggerSkipPaths = []string{
	HealthPath,
}

type Server struct {
	group     *gin.RouterGroup
	router    *gin.Engine
	apiServer *gorest.Server
}

type HandlerRegister func(*gorest.Server, gin.IRoutes) error

func (h HandlerRegister) RegisterHandler(server *gorest.Server, router gin.IRoutes) error {
	return h(server, router)
}

type WebHandler interface {
	RegisterHandler(*gorest.Server, gin.IRoutes) error
}

func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = os.Stdout
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: LoggerSkipPaths,
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[%s] client:%s \"%s %s\" %s %d %s %s\n",
				param.TimeStamp.Format(excel.TimeFormat),
				param.ClientIP,
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
			)
		},
	}))

	router.GET(HealthPath, HealthCheck)
	excel.RegisterFileApi(router, dhcp.Version.GetUrl())
	group := router.Group("/")
	apiServer := gorest.NewAPIServer(schema.NewSchemaManager())
	return &Server{
		group:     group,
		router:    router,
		apiServer: apiServer,
	}, nil
}

func (s *Server) RegisterHandler(h WebHandler) error {
	return h.RegisterHandler(s.apiServer, s.router)
}

func (s *Server) Run(conf *config.DHCPConfig) error {
	errch := make(chan error)
	adaptor.RegisterHandler(s.group, s.apiServer, s.apiServer.Schemas.GenerateResourceRoute())

	if register, err := RegisterHttp(conf, config.ConsulConfig); err != nil {
		return fmt.Errorf("register consul failed: %s", err.Error())
	} else {
		if err := register.Register(); err != nil {
			return fmt.Errorf("register consul http service failed: %s", err.Error())
		} else {
			defer register.Deregister()
		}
	}

	if register, err := RegisterGrpc(conf, config.ConsulConfig); err != nil {
		return fmt.Errorf("register consul failed: %s", err.Error())
	} else {
		if err := register.Register(); err != nil {
			return fmt.Errorf("register consul grpc service failed: %s", err.Error())
		} else {
			defer register.Deregister()
		}
	}

	go func() {
		errch <- s.router.Run(fmt.Sprintf(":%d", conf.Server.Port))
	}()

	go func() {
		grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.Server.GrpcPort))
		if err != nil {
			errch <- err
			return
		}

		grpcServer := grpc.NewServer(
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle: 5 * time.Minute,
				Time:              time.Second * 30,
				Timeout:           time.Second * 5,
			}),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             time.Second * 30,
				PermitWithoutStream: true,
			}))
		hv1.RegisterHealthServer(grpcServer, health.NewServer())
		pbdhcp.RegisterDhcpServiceServer(grpcServer, service.NewGrpcService())
		errch <- grpcServer.Serve(grpcListener)
	}()

	err := <-errch
	return err
}
