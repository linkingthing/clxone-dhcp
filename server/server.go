package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zdnscloud/gorest"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/resource/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	hv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/transports"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
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
	gin.SetMode(gin.DebugMode)
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

	router.GET("/health", func(context *gin.Context) {
		context.Writer.Header().Set("Content-Type", "Application/Json")
		context.String(http.StatusOK, `{"status": "ok"}`)
	})
	group := router.Group("/")
	apiServer := gorest.NewAPIServer(schema.NewSchemaManager())
	apiServer.Use(JWTMiddleWare())
	return &Server{
		group:     group,
		router:    router,
		apiServer: apiServer,
	}, nil
}

func (s *Server) RegisterHandler(h WebHandler) error {
	return h.RegisterHandler(s.apiServer, s.router)
}

func (s *Server) Run(conf *config.DHCPConfig) (err error) {
	errc := make(chan error)
	adaptor.RegisterHandler(s.group, s.apiServer, s.apiServer.Schemas.GenerateResourceRoute())

	{
		// register rest api service to consul
		serviceName := "clxone-dhcp-api"
		serviceID := serviceName + uuid.NewString()
		registar := RegisterForHttp(conf.Server.IP,
			conf.Server.Port,
			serviceID,
			serviceName,
		)
		defer registar.Deregister()
	}

	{
		// register grpc api service to consul
		grpcServiceName := "clxone-dhcp-grpc"
		grpcServiceID := grpcServiceName + uuid.NewString()
		registar := RegisterForGrpc(conf.Server.IP,
			getGrpcPort(conf.Server.Port),
			grpcServiceID,
			grpcServiceName,
		)
		registar.Register()
		defer registar.Deregister()
	}

	{
		go func() {
			errc <- s.router.Run(fmt.Sprintf(":%d", conf.Server.Port))
		}()

		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
			errc <- fmt.Errorf("%s", <-c)
		}()

		go func() {
			grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", getGrpcPort(conf.Server.Port)))
			if err != nil {
				errc <- err
				return
			}
			baseServer := grpc.NewServer()
			healthServer := health.NewServer()
			hv1.RegisterHealthServer(baseServer, healthServer)

			dhcp.RegisterDhcpServiceServer(baseServer,
				transports.DHCPServiceBinding{DHCPService: services.NewDHCPService()})

			errc <- baseServer.Serve(grpcListener)
		}()
	}

	err = <-errc
	return err
}

func getGrpcPort(httpPort int) (grpcPort int) {
	return httpPort + 1
}
