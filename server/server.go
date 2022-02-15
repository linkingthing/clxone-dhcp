package server

import (
	"fmt"

	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
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
	csvutil "github.com/linkingthing/clxone-utils/csv"
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
				param.TimeStamp.Format(csvutil.TimeFormat),
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

	router.GET(HealthPath, func(context *gin.Context) {
		context.Writer.Header().Set("Content-Type", "Application/Json")
		context.String(http.StatusOK, `{"status": "ok"}`)
	})
	csvutil.RegisterFileApi(router, dhcp.Version.GetUrl())
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
	if apiRegister, err := NewHttpRegister(consulapi.AgentServiceRegistration{
		ID:      conf.Consul.Name + "-api-" + conf.Server.IP,
		Name:    conf.Consul.Name + "-api",
		Address: conf.Server.IP,
		Port:    conf.Server.Port,
	}); err != nil {
		return err
	} else {
		if err := apiRegister.Register(); err != nil {
			return err
		} else {
			defer apiRegister.Deregister()
		}
	}

	if grpcRegister, err := NewGrpcRegister(
		consulapi.AgentServiceRegistration{
			ID:      conf.Consul.Name + "-grpc-" + conf.Server.IP,
			Name:    conf.Consul.Name + "-grpc",
			Address: conf.Server.IP,
			Port:    conf.Server.GrpcPort,
		}); err != nil {
		return err
	} else {
		if err := grpcRegister.Register(); err != nil {
			return err
		} else {
			defer grpcRegister.Deregister()
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

		grpcServer := grpc.NewServer()
		hv1.RegisterHealthServer(grpcServer, health.NewServer())
		pbdhcp.RegisterDhcpServiceServer(grpcServer, service.NewGRPCService())
		errch <- grpcServer.Serve(grpcListener)
	}()

	err := <-errch
	return err
}
