package resource

import (
	"strings"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"

	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Node struct {
	resource.ResourceBase `json:",inline"`

	Ip           string               `json:"ip"`
	Ipv6s        []string             `json:"ipv6s"`
	Macs         []string             `json:"macs"`
	Roles        []string             `json:"roles"`
	HostName     string               `json:"hostName"`
	NodeIsAlive  bool                 `json:"nodeIsAlive"`
	DhcpIsAlive  bool                 `json:"dhcpIsAlive"`
	DnsIsAlive   bool                 `json:"dnsIsAlive"`
	CpuUsage     []RatioWithTimestamp `json:"cpuUsage" db:"-"`
	MemoryUsage  []RatioWithTimestamp `json:"memoryUsage" db:"-"`
	DiscUsage    []RatioWithTimestamp `json:"discUsage" db:"-"`
	Network      []RatioWithTimestamp `json:"network" db:"-"`
	CpuRatio     string               `json:"cpuRatio"`
	MemRatio     string               `json:"memRatio"`
	Master       string               `json:"master"`
	ControllerIp string               `json:"controllerIP"`
	StartTime    time.Time            `json:"startTime"`
	UpdateTime   time.Time            `json:"updateTime"`
	Vip          string               `json:"vip"`
}

const (
	ActionMasterUp   = "master_up"
	ActionMasterDown = "master_down"
	ActionRegister   = "register"
	ActionKeepalive  = "keepalive"
)

func (node Node) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:  ActionMasterUp,
			Input: &HaRequest{},
		},
		resource.Action{
			Name:  ActionMasterDown,
			Input: &HaRequest{},
		},
		resource.Action{
			Name:  ActionRegister,
			Input: &Node{},
		},
		resource.Action{
			Name:  ActionKeepalive,
			Input: &Node{},
		},
	}
}

func GenESDNSIndex(nodes []*Node) string {
	var dnsIndexes []string
	for _, node := range nodes {
		if slice.SliceIndex(node.Roles, string(ServiceRoleDNS)) != -1 {
			dnsIndexes = append(dnsIndexes, "dns_"+node.Ip)
		}
	}

	return strings.Join(dnsIndexes, ",")
}

func IsMaster(serverIp string) (bool, error) {
	isMaster := false
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if master_, err := IsMasterWithTx(serverIp, tx); err != nil {
			return err
		} else {
			isMaster = master_
		}
		return nil
	}); err != nil {
		return isMaster, err
	}

	return isMaster, nil
}

func IsMasterWithTx(serverIp string, tx restdb.Transaction) (bool, error) {
	var nodes []*Node
	sql := util.GenSqlQueryArray(restdb.ResourceDBType(&Node{}), "roles", string(ServiceRoleController)) +
		" and id ='" + serverIp + "' and master='' "
	if err := tx.FillEx(&nodes, sql); err != nil {
		return false, err
	}

	if len(nodes) == 1 {
		return true, nil
	}

	return false, nil
}

func GetLocalIPFromDB() (string, error) {
	vip := config.GetConfig().Server.IP
	var nodes []*Node
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: config.GetConfig().Server.IP}, &nodes)
	}); err != nil {
		return vip, err
	}

	for _, node := range nodes {
		vip = node.Vip
	}

	return vip, nil
}

func GenDnsIndex() (string, error) {
	var nodes []*Node
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &nodes)
	}); err != nil {
		return "", err
	}

	return GenESDNSIndex(nodes), nil
}
