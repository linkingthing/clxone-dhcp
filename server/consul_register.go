package server

import (
	"fmt"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/linkingthing/clxone-dhcp/config"
)

func RegisterForHttp(
	advertiseAddress string,
	advertisePort int,
	serviceID string,
	serviceName string) (registar sd.Registrar) {

	check := consulapi.AgentServiceCheck{
		HTTP:     fmt.Sprintf("http://%v:%v/health", advertiseAddress, advertisePort),
		Interval: config.GetConfig().Consul.Check.Interval,
		Timeout:  config.GetConfig().Consul.Check.Timeout,
	}

	registar = register(advertiseAddress,
		advertisePort,
		serviceID,
		serviceName,
		check)
	return registar
}

func RegisterForGrpc(
	advertiseAddress string,
	advertisePort int,
	serviceID string,
	serviceName string) (registar sd.Registrar) {

	check := consulapi.AgentServiceCheck{
		GRPC:     fmt.Sprintf("%v:%v", advertiseAddress, advertisePort),
		Interval: config.GetConfig().Consul.Check.Interval,
		Timeout:  config.GetConfig().Consul.Check.Timeout,
	}

	registar = register(advertiseAddress,
		advertisePort,
		serviceID,
		serviceName,
		check)
	return registar
}

func register(advertiseAddress string,
	advertisePort int,
	serviceID string,
	serviceName string,
	check consulapi.AgentServiceCheck) (registar sd.Registrar) {

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	consulClient, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	asr := consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: advertiseAddress,
		Port:    advertisePort,
		Check:   &check,
	}
	client := consulsd.NewClient(consulClient)
	registar = consulsd.NewRegistrar(client, &asr, logger)
	return
}
