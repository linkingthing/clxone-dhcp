package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
	consulutil "github.com/linkingthing/clxone-utils/consul"

	"github.com/linkingthing/clxone-dhcp/config"
)

func RegisterHttp(conf *config.DHCPConfig, consulConfig *consulapi.Config) (*consulutil.Registrar, error) {
	register, err := consulutil.RegisterHttp(consulConfig, consulapi.AgentServiceRegistration{
		ID:   conf.Consul.HttpName + "-" + conf.Server.IP,
		Name: conf.Consul.HttpName,
		Port: conf.Server.Port,
		Tags: conf.Consul.Tags,
	})

	if err != nil {
		return register, err
	}

	return register, nil
}

func RegisterGrpc(conf *config.DHCPConfig, consulConfig *consulapi.Config) (*consulutil.Registrar, error) {
	register, err := consulutil.RegisterGrpc(consulConfig, consulapi.AgentServiceRegistration{
		ID:   conf.Consul.GrpcName + "-" + conf.Server.IP,
		Name: conf.Consul.GrpcName,
		Port: conf.Server.GrpcPort,
		Tags: conf.Consul.Tags,
	})

	if err != nil {
		return register, err
	}

	return register, nil
}

func HealthCheck(ctx *gin.Context) {
	ctx.String(http.StatusOK, "health check ok")
}
