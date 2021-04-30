package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/zdnscloud/gorest"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/resource/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	hv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	defaultTlsCertFile = "tls_cert.crt"
	defaultTlsKeyFile  = "tls_key.key"
)

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
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] client:%s \"%s %s\" %s %d %s %s\n",
			param.TimeStamp.Format(util.TimeFormat),
			param.ClientIP,
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
		)
	}))

	group := router.Group("/")
	apiServer := gorest.NewAPIServer(schema.NewSchemaManager())
	// apiServer.Use(authentification.JWTMiddleWare())
	return &Server{
		group:     group,
		router:    router,
		apiServer: apiServer,
	}, nil
}

func (s *Server) RegisterHandler(h WebHandler) error {
	return h.RegisterHandler(s.apiServer, s.router)
}

func (s *Server) Run(conf *config.DDIControllerConfig) (err error) {
	errc := make(chan error)
	adaptor.RegisterHandler(s.group, s.apiServer, s.apiServer.Schemas.GenerateResourceRoute())

	{
		// register grpc api service to consul
		check := consulapi.AgentServiceCheck{
			GRPC:     fmt.Sprintf("%s:%s", conf.Server.IP, conf.Server.GrpcPort),
			Interval: "10s",
			Timeout:  "1s",
		}
		grpcServiceName := "clxone-dhcp-grpc"
		grpcServiceID := grpcServiceName + uuid.NewString()
		registar := Register(conf.Server.IP, conf.Server.GrpcPort, grpcServiceID, grpcServiceName, check)
		registar.Register()
		defer registar.Deregister()
	}

	{
		// register rest api service to consul
		check := consulapi.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%v:%v/health", conf.Server.IP, conf.Server.Port),
			Interval: "10s",
			Timeout:  "1s",
		}
		serviceName := "clxone-dhcp-api"
		serviceID := serviceName + uuid.NewString()
		registar := Register(conf.Server.IP, conf.Server.Port, serviceID, serviceName, check)
		defer registar.Deregister()
	}

	{
		go func() {
			errc <- s.router.Run(":" + conf.Server.Port)
		}()

		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			errc <- fmt.Errorf("%s", <-c)
		}()

		go func() {
			grpcListener, err := net.Listen("tcp", ":"+conf.Server.GrpcPort)
			if err != nil {
				errc <- err
				return
			}
			baseServer := grpc.NewServer()
			healthServer := health.NewServer()
			hv1.RegisterHealthServer(baseServer, healthServer)
			// svc := service.UserService{}
			// pb.RegisterUserServiceServer(baseServer, transport.UserServiceBinding{UserService: svc})

			errc <- baseServer.Serve(grpcListener)
		}()
	}

	err = <-errc
	return err
}
