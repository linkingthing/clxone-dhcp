package config

import (
	"github.com/zdnscloud/cement/configure"
)

type DDIControllerConfig struct {
	Path                  string             `yaml:"-"`
	DB                    DBConf             `yaml:"db"`
	Server                ServerConf         `yaml:"server"`
	Kafka                 KafkaConf          `yaml:"kafka"`
	DDIAgent              DDIAgentConf       `yaml:"ddi_agent"`
	Prometheus            PrometheusConf     `yaml:"prometheus"`
	Elasticsearch         ElasticsearchConf  `yaml:"elasticsearch"`
	MonitorNode           MonitorNodeConf    `yaml:"monitor_node"`
	AuditLog              AuditLogConf       `yaml:"audit_log"`
	Alarm                 AlarmConf          `yaml:"alarm"`
	SubnetScan            SubnetScanConf     `yaml:"subnet_scan"`
	IllegalDHCPServerScan DHCPServerScanConf `yaml:"illegal_dhcp_server_scan"`
	RegionData            RegionDataConf     `yaml:"region_data"`
}

type DBConf struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
	Host     string `json:"host"`
}

type ServerConf struct {
	IP          string `yaml:"ip"`
	Port        int    `yaml:"port"`
	Hostname    string `yaml:"hostname"`
	TlsCertFile string `yaml:"tls_cert_file"`
	TlsKeyFile  string `yaml:"tls_key_file"`
	Master      string `yaml:"master"`
	NotifyAddr  string `yaml:"notify_addr"`
}

type DDIAgentConf struct {
	GrpcAddr string `yaml:"grpc_addr"`
}

type KafkaConf struct {
	Addr              []string `yaml:"kafka_addrs"`
	GroupIdAgentEvent string   `yaml:"group_id_agentevent"`
	GroupIdUploadLog  string   `yaml:"group_id_uploadlog"`
}

type PrometheusConf struct {
	Addr       string `yaml:"addr"`
	ExportPort int    `yaml:"export_port"`
}

type MonitorNodeConf struct {
	TimeOut int64 `yaml:"time_out"`
}

type ElasticsearchConf struct {
	Addr  []string `yaml:"es_addrs"`
	Index string   `yaml:"index"`
}

type AlarmConf struct {
	ValidPeriod uint32 `yaml:"valid_period"`
}

type AuditLogConf struct {
	ValidPeriod uint32 `yaml:"valid_period"`
}

type SubnetScanConf struct {
	Interval uint32 `yaml:"interval"`
}

type DHCPServerScanConf struct {
	Interval uint32 `yaml:"interval"`
}

type RegionDataConf struct {
	ProvinceData string `yaml:"province_data"`
	CityData     string `yaml:"city_data"`
}

var gConf *DDIControllerConfig

func LoadConfig(path string) (*DDIControllerConfig, error) {
	var conf DDIControllerConfig
	conf.Path = path
	if err := conf.Reload(); err != nil {
		return nil, err
	}

	return &conf, nil
}

func (c *DDIControllerConfig) Reload() error {
	var newConf DDIControllerConfig
	if err := configure.Load(&newConf, c.Path); err != nil {
		return err
	}

	newConf.Path = c.Path
	*c = newConf
	gConf = &newConf
	return nil
}

func GetConfig() *DDIControllerConfig {
	return gConf
}
