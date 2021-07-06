package server

import (
	"github.com/go-kit/kit/sd/consul"
	stdconsul "github.com/hashicorp/consul/api"
)

// Registrar registers service instance liveness information to Consul.
type Registrar struct {
	client       consul.Client
	registration *stdconsul.AgentServiceRegistration
}

// NewRegistrar returns a Consul Registrar acting on the provided catalog
// registration.
func NewRegistrar(client consul.Client, r *stdconsul.AgentServiceRegistration) *Registrar {
	return &Registrar{
		client:       client,
		registration: r,
	}
}

// Register implements sd.Registrar interface.
func (p *Registrar) Register() error {
	return p.client.Register(p.registration)
}

// Deregister implements sd.Registrar interface.
func (p *Registrar) Deregister() error {
	return p.client.Deregister(p.registration)
}
