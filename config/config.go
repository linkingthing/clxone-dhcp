package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/linkingthing/cement/configure"
	"github.com/linkingthing/clxone-utils/pbe"
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
	IP              string `yaml:"ip"`
	Port            int    `yaml:"port"`
	GrpcPort        int    `yaml:"grpc_port"`
	Hostname        string `yaml:"hostname"`
	KeyFactoryPath  string `yaml:"key_factory_path"`
	WorkKeyPath     string `yaml:"work_key_path"`
	DecryptIterator int    `yaml:"decrypt_iterator"`
}

type KafkaConf struct {
	Addrs                     []string `yaml:"kafka_addrs"`
	GroupUpdateThresholdEvent string   `yaml:"group_id_update_threshold_event"`
	Username                  string   `yaml:"username"`
	Password                  string   `yaml:"password"`
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
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	CertPem    []byte `yaml:"-"`
	KeyPem     []byte `yaml:"-"`
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
	err := configure.Load(&newConf, c.Path)
	if err != nil {
		return err
	}

	if err = newConf.parsePrometheusTlsConfig(); err != nil {
		return err
	}

	if newConf.DB.Password, err = decryptPassword(newConf.DB.Password, newConf.Server); err != nil {
		return err
	}

	if newConf.Kafka.Password, err = decryptPassword(newConf.Kafka.Password, newConf.Server); err != nil {
		return err
	}

	if newConf.Prometheus.Password, err = decryptPassword(newConf.Prometheus.Password, newConf.Server); err != nil {
		return err
	}

	newConf.Path = c.Path
	*c = newConf
	gConf = &newConf
	return nil
}

func decryptPassword(encryptPassword string, conf ServerConf) (string, error) {
	keyFactoryBase64, err := readConfFromFile(conf.KeyFactoryPath)
	if err != nil {
		return "", err
	}

	encryptWorkKey, err := readConfFromFile(conf.WorkKeyPath)
	if err != nil {
		return "", err
	}

	iterator := conf.DecryptIterator
	if iterator == 0 {
		iterator = 10000
	}

	return pbe.Decrypt(&pbe.DecryptContext{
		KeyFactoryBase64: keyFactoryBase64,
		EncryptWorkKey:   encryptWorkKey,
		EncryptPassword:  encryptPassword,
		Iterator:         iterator,
	})
}

func readConfFromFile(path string) (string, error) {
	if content, err := ioutil.ReadFile(path); err != nil {
		return "", err
	} else {
		return strings.TrimRight(string(content), "\r\n"), nil
	}
}

func GetConfig() *DHCPConfig {
	return gConf
}

func (c *DHCPConfig) parsePrometheusTlsConfig() error {
	if keyPem, err := ioutil.ReadFile(c.Prometheus.KeyFile); err != nil {
		return fmt.Errorf("read prometheus key file failed:%s", err.Error())
	} else {
		c.Prometheus.KeyPem = keyPem
	}

	if certPem, err := ioutil.ReadFile(c.Prometheus.CertFile); err != nil {
		return fmt.Errorf("read prometheus cert file failed:%s", err.Error())
	} else {
		c.Prometheus.CertPem = certPem
	}

	return nil
}
