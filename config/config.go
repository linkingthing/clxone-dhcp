package config

import (
	"github.com/linkingthing/cement/configure"
)

type DHCPConfig struct {
	Path         string          `yaml:"-"`
	DB           DBConf          `yaml:"db"`
	Server       ServerConf      `yaml:"server"`
	Kafka        KafkaConf       `yaml:"kafka"`
	Prometheus   PrometheusConf  `yaml:"prometheus"`
	Consul       ConsulConf      `yaml:"consul"`
	CallServices CallServiceConf `yaml:"call_services"`
	DHCPScan     DHCPScanConf    `yaml:"dhcp_scan"`
}

type DBConf struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
	Host     string `json:"host"`
}

type ServerConf struct {
	IP       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	GrpcPort int    `yaml:"grpc_port"`
	Hostname string `yaml:"hostname"`
}

type KafkaConf struct {
	Addrs                     []string `yaml:"kafka_addrs"`
	GroupUpdateThresholdEvent string   `yaml:"group_id_update_threshold_event"`
}

type CallServiceConf struct {
	Logging   string `yaml:"logging"`
	User      string `yaml:"user"`
	Ipam      string `yaml:"ipam"`
	Dns       string `yaml:"dns"`
	Dhcp      string `yaml:"dhcp"`
	DnsAgent  string `yaml:"dns-agent"`
	DhcpAgent string `yaml:"dhcp-agent"`
	Boxsearch string `yaml:"boxsearch"`
	Monitor   string `yaml:"monitor"`
	Alarm     string `yaml:"alarm"`
}

type DHCPScanConf struct {
	Interval uint32 `yaml:"interval"`
}
type PrometheusConf struct {
	Addr       string `yaml:"addr"`
	ExportPort int    `yaml:"export_port"`
}

type ConsulConf struct {
	Address string    `yaml:"address"`
	ID      string    `yaml:"id"`
	Name    string    `yaml:"name"`
	Tags    []string  `yaml:"tags"`
	Check   CheckConf `yaml:"check"`
}

type CheckConf struct {
	Interval                       string `yaml:"interval"`
	Timeout                        string `yaml:"timeout"`
	DeregisterCriticalServiceAfter string `yaml:"deregister_critical_service_after"`
	TLSSkipVerify                  bool   `yaml:"tls_skip_verify"`
}

var gConf *DHCPConfig

func LoadConfig(path string) (*DHCPConfig, error) {
	var conf DHCPConfig
	conf.Path = path
	if err := conf.Reload(); err != nil {
		return nil, err
	}

	return &conf, nil
}

func (c *DHCPConfig) Reload() error {
	var newConf DHCPConfig
	if err := configure.Load(&newConf, c.Path); err != nil {
		return err
	}

	newConf.Path = c.Path
	*c = newConf
	gConf = &newConf
	return nil
}

func GetConfig() *DHCPConfig {
	return gConf
}
