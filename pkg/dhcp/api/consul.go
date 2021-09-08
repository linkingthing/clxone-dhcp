package api

import (
	"fmt"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/util/httpclient"
)

const (
	ChecksUrl   = "http://%s/v1/agent/checks?filter=ServiceName==\"%s\""
	ServicesUrl = "http://%s/v1/catalog/service/%s"
)

type ConsulServiceStatus string

const (
	ConsulServiceStatusPassing  ConsulServiceStatus = "passing"
	ConsulServiceStatusWarning  ConsulServiceStatus = "warning"
	ConsulServiceStatusCritical ConsulServiceStatus = "critical"
)

type ConsulService struct {
	Address     string              `json:"Address"`
	ServiceID   string              `json:"ServiceID"`
	ServiceName string              `json:"ServiceName"`
	ServiceTags []string            `json:"ServiceTags"`
	Status      ConsulServiceStatus `json:"Status"`
}

func (cs *ConsulService) Validate() bool {
	return cs.Status == ConsulServiceStatusPassing
}

type ConsulHandler struct {
	consulChecksUrl   string
	consulServicesUrl string
}

var gConsulHandler *ConsulHandler

func InitConsulHandler() {
	gConsulHandler = &ConsulHandler{
		consulChecksUrl: fmt.Sprintf(ChecksUrl, config.GetConfig().Consul.Address,
			config.GetConfig().CallServices.DhcpAgent),
		consulServicesUrl: fmt.Sprintf(ServicesUrl, config.GetConfig().Consul.Address,
			config.GetConfig().CallServices.DhcpAgent),
	}
}

func GetConsulHandler() *ConsulHandler {
	return gConsulHandler
}

func (h *ConsulHandler) GetDHCPAgentChecksAndServices() (map[string]*ConsulService, []*ConsulService, error) {
	var checks map[string]*ConsulService
	if err := httpclient.GetHttpClient().Get(h.consulChecksUrl, &checks); err != nil {
		return nil, nil, fmt.Errorf("list dhcp agent checks failed: %s", err.Error())
	}

	var services []*ConsulService
	if err := httpclient.GetHttpClient().Get(h.consulServicesUrl, &services); err != nil {
		return nil, nil, fmt.Errorf("list dhcp agents services failed: %s", err.Error())
	}

	return checks, services, nil
}
