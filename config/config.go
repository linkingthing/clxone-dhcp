package config

import (
	"github.com/zdnscloud/cement/configure"
)

type DHCPConfig struct {
	Path          string            `yaml:"-"`
	DB            DBConf            `yaml:"db"`
	Server        ServerConf        `yaml:"server"`
	Kafka         KafkaConf         `yaml:"kafka"`
	Prometheus    PrometheusConf    `yaml:"prometheus"`
	Elasticsearch ElasticsearchConf `yaml:"elasticsearch"`
	Consul        ConsulConf        `yaml:"consul"`
	CallServices  map[string]string `yaml: "call_services"`
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
	GrpcPort    int    `yaml:"grpc_port"`
	Hostname    string `yaml:"hostname"`
	TlsCertFile string `yaml:"tls_cert_file"`
	TlsKeyFile  string `yaml:"tls_key_file"`
	Master      string `yaml:"master"`
	NotifyAddr  string `yaml:"notify_addr"`
}

type KafkaConf struct {
	Addr                      []string `yaml:"kafka_addrs"`
	GroupIdAgentEvent         string   `yaml:"group_id_agentevent"`
	GroupIdUploadLog          string   `yaml:"group_id_uploadlog"`
	GroupUpdateThresholdEvent string   `yaml:"group_id_update_threshold_event"`
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

type ConsulConf struct {
	ID    string    `yaml:"id"`
	Name  string    `yaml:"name"`
	Tags  []string  `yaml:"tags"`
	Check CheckConf `yaml:"check"`
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
