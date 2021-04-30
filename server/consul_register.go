package server

import (
	"os"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	consulapi "github.com/hashicorp/consul/api"
)

func Register(advertiseAddress string,
	advertisePort string,
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

	port, _ := strconv.Atoi(advertisePort)
	asr := consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: advertiseAddress,
		Port:    port,
		Check:   &check,
	}
	client := consulsd.NewClient(consulClient)
	registar = consulsd.NewRegistrar(client, &asr, logger)
	registar.Register()
	return
}
