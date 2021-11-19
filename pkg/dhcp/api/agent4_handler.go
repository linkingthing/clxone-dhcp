package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type AgentRole string

const (
	AgentRoleSentry4 AgentRole = "sentry4"
	AgentRoleServer4 AgentRole = "server4"
	AgentRoleSentry6 AgentRole = "sentry6"
	AgentRoleServer6 AgentRole = "server6"
)

type Agent4Handler struct {
}

func NewAgent4Handler() *Agent4Handler {
	return &Agent4Handler{}
}

func (h *Agent4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	checks, services, err := GetConsulHandler().GetDHCPAgentChecksAndServices()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agents failed: %s", err.Error()))
	}

	var agents []*resource.Agent4
	for _, check := range checks {
		if check.Validate() {
			if service := getSentryServiceWithServiceID(check.ServiceID, services,
				AgentRoleSentry4); service != nil {
				agent := &resource.Agent4{Ip: service.ServiceAddress}
				agent.SetID(service.ServiceAddress)
				agents = append(agents, agent)
			}
		}
	}

	return agents, nil
}

func getSentryServiceWithServiceID(id string, services []*ConsulService, role ...AgentRole) *ConsulService {
	for _, service := range services {
		if service.ServiceID == id && isAgentServiceMatchRoles(service, role...) {
			return service
		}
	}

	return nil
}

func isAgentServiceMatchRoles(service *ConsulService, roles ...AgentRole) bool {
	for _, tag := range service.ServiceTags {
		for _, role := range roles {
			if tag == string(role) {
				return true
			}
		}
	}

	return false
}

func (h *Agent4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	agent := ctx.Resource.(*resource.Agent4)
	checks, services, err := GetConsulHandler().GetDHCPAgentChecksAndServices()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agents failed: %s", err.Error()))
	}

	for _, check := range checks {
		if check.Validate() {
			if service := getSentryServiceWithServiceID(check.ServiceID, services,
				AgentRoleSentry4); service != nil &&
				service.ServiceAddress == agent.GetID() {
				agent.Ip = service.ServiceAddress
				return agent, nil
			}
		}
	}

	return nil, resterror.NewAPIError(resterror.NotFound,
		fmt.Sprintf("no found dhcp agent %s", agent.GetID()))
}
