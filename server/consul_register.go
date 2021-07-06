package server

import (
	"fmt"

	consulsd "github.com/go-kit/kit/sd/consul"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/linkingthing/clxone-dhcp/config"
)

func NewHttpRegister(registration consulapi.AgentServiceRegistration) (*Registrar, error) {
	return register(registration, consulapi.AgentServiceCheck{
		HTTP:                           fmt.Sprintf("http://%v:%v/health", registration.Address, registration.Port),
		Interval:                       config.GetConfig().Consul.Check.Interval,
		Timeout:                        config.GetConfig().Consul.Check.Timeout,
		DeregisterCriticalServiceAfter: config.GetConfig().Consul.Check.DeregisterCriticalServiceAfter,
		TLSSkipVerify:                  config.GetConfig().Consul.Check.TLSSkipVerify,
	})
}

func NewGrpcRegister(registration consulapi.AgentServiceRegistration) (*Registrar, error) {
	return register(registration, consulapi.AgentServiceCheck{
		GRPC:                           fmt.Sprintf("%v:%v", registration.Address, registration.Port),
		Interval:                       config.GetConfig().Consul.Check.Interval,
		Timeout:                        config.GetConfig().Consul.Check.Timeout,
		DeregisterCriticalServiceAfter: config.GetConfig().Consul.Check.DeregisterCriticalServiceAfter,
		TLSSkipVerify:                  config.GetConfig().Consul.Check.TLSSkipVerify,
	})
}

func register(registration consulapi.AgentServiceRegistration, check consulapi.AgentServiceCheck) (*Registrar, error) {
	conf := consulapi.DefaultConfig()
	conf.Address = config.GetConfig().Consul.Address
	consulClient, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("new consul client failed: %s", err.Error())
	}

	registration.Tags = config.GetConfig().Consul.Tags
	registration.Checks = consulapi.AgentServiceChecks{&check}
	return NewRegistrar(consulsd.NewClient(consulClient), &registration), nil
}
