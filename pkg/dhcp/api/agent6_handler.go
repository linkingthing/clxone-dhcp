package api

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type Agent6Handler struct {
}

func NewAgent6Handler() *Agent6Handler {
	return &Agent6Handler{}
}

func (h *Agent6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	checks, services, err := GetConsulHandler().GetDHCPAgentChecksAndServices()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agent6s failed: %s", err.Error()))
	}

	var agents []*resource.Agent6
	for _, check := range checks {
		if check.Validate() {
			if service := getSentryServiceWithServiceID(check.ServiceID, services,
				isAgentServiceMatchRoles, AgentRoleSentry6); service != nil {
				agent := &resource.Agent6{Ip: service.Address}
				agent.SetID(service.Address)
				agents = append(agents, agent)
				break
			}
		}
	}

	return agents, nil
}

func (h *Agent6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	agent := ctx.Resource.(*resource.Agent6)
	checks, services, err := GetConsulHandler().GetDHCPAgentChecksAndServices()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agent6s failed: %s", err.Error()))
	}

	for _, check := range checks {
		if check.Validate() {
			if service := getSentryServiceWithServiceID(check.ServiceID, services,
				isAgentServiceMatchRoles, AgentRoleSentry6); service != nil && service.Address == agent.GetID() {
				agent.Ip = service.Address
				return agent, nil
			}
		}
	}

	return nil, resterror.NewAPIError(resterror.NotFound,
		fmt.Sprintf("no found dhcp agent6 %s", agent.GetID()))
}
