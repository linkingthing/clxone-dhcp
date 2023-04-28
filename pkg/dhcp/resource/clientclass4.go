package resource

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type OptionCondition string

const (
	OptionConditionExists         OptionCondition = "exists"
	OptionConditionEqual          OptionCondition = "equal"
	OptionConditionSubstringEqual OptionCondition = "substring"
)

var TableClientClass4 = restdb.ResourceDBType(&ClientClass4{})

type ClientClass4 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string          `json:"name" rest:"required=true,description=immutable" db:"uk"`
	Code                      uint8           `json:"code" rest:"required=true,description=immutable"`
	Condition                 OptionCondition `json:"condition" rest:"required=true,options=exists|equal|substring"`
	Regexp                    string          `json:"regexp"`
	BeginIndex                uint32          `json:"beginIndex"`
	Description               string          `json:"description"`
}

var ErrNameOrRegexpMissing = fmt.Errorf("clientclass name and regexp are required")

func (c *ClientClass4) Validate() error {
	if len(c.Name) == 0 || (c.Condition != OptionConditionExists && len(c.Regexp) == 0) {
		return ErrNameOrRegexpMissing
	} else if c.Code < 1 || c.Code > 254 {
		return fmt.Errorf("code %d not in [1, 254]", c.Code)
	} else if err := util.ValidateStrings(util.RegexpTypeCommon, c.Name, c.Description); err != nil {
		return fmt.Errorf("name %s or description %s is invalid", c.Name, c.Description)
	} else if err := util.ValidateStrings(util.RegexpTypeSlash, c.Regexp); err != nil {
		return fmt.Errorf("regexp %s is invalid", c.Regexp)
	} else {
		if c.Description == "" {
			c.Description = code4ToDescription(c.Code)
		}

		return nil
	}
}

var code4Description = map[uint8]string{
	0:   "pad",
	1:   "subnet-mask",
	2:   "time-offset",
	3:   "router",
	4:   "time-server",
	5:   "name-server",
	6:   "domain-name-server",
	7:   "log-server",
	8:   "quotes-server",
	9:   "lpr-server",
	10:  "impress-server",
	11:  "rlp-server",
	12:  "hostname",
	13:  "boot-file-size",
	14:  "merit-dump-file",
	15:  "domain-name",
	16:  "swap-server",
	17:  "root-path",
	18:  "extension-file",
	19:  "ip-forwarding",
	20:  "non-local-source-routing",
	21:  "policy-filter",
	22:  "max-datagram-reassembly-size",
	23:  "default-ip-ttl",
	24:  "path-mtu-aging-timeout",
	25:  "path-mtu-plateau-table",
	26:  "interface-mtu",
	27:  "all-subnets-are-local",
	28:  "broadcast-address",
	29:  "perform-mask-discovery",
	30:  "mask-supplier",
	31:  "perform-router-discovery",
	32:  "router-solicitation-address",
	33:  "static-routing-table",
	34:  "trailer-encapsulation",
	35:  "arp-cache-timeout",
	36:  "ethernet-encapsulation",
	37:  "defaul-tcp-ttl",
	38:  "tcp-keepalive-interval",
	39:  "tcp-keepalive-garbage",
	40:  "network-information-service-domain",
	41:  "network-information-servers",
	42:  "ntp-servers",
	43:  "vendor-specific-information",
	44:  "netbios-over-tcp-ip-name-server",
	45:  "netbios-over-tcp-ip-datagram-distribution-server",
	46:  "netbios-over-tcp-ip-node-type",
	47:  "netbios-over-tcp-ip-scope",
	48:  "x-window-system-font-server",
	49:  "x-window-system-display-manger",
	50:  "requested-ip-address",
	51:  "ip-address-lease-time",
	52:  "option-overload",
	53:  "dhcp-message-type",
	54:  "server-identifier",
	55:  "parameter-request-list",
	56:  "message",
	57:  "maximum-dhcp-message-size",
	58:  "renew-time-value",
	59:  "rebinding-time-value",
	60:  "class-identifier",
	61:  "client-identifier",
	62:  "netware-ip-domain-name",
	63:  "netware-ip-information",
	64:  "network-information-service-plus-domain",
	65:  "network-information-service-plus-servers",
	66:  "tftp-server-name",
	67:  "boot-file-name",
	68:  "mobile-ip-home-agent",
	69:  "smtp-server",
	70:  "pop-server",
	71:  "nntp-server",
	72:  "default-www-server",
	73:  "default-finger-server",
	74:  "default-irc-server",
	75:  "street-talk-server",
	76:  "street-talk-directory-assistance-server",
	77:  "user-class-information",
	78:  "slp-directory-agent",
	79:  "slp-service-scope",
	80:  "rapid-commit",
	81:  "fqdn",
	82:  "relay-agent-information",
	83:  "internet-storage-name-service",
	85:  "nds-servers",
	86:  "nds-tree-name",
	87:  "nds-context",
	88:  "bcmcs-controller-domain-name-list",
	89:  "bcmcs-controller-ipv4-address-list",
	90:  "authentication",
	91:  "client-last-transaction-time",
	92:  "associated-ip",
	93:  "client-system-architecture-type",
	94:  "client-network-interface-identifier",
	95:  "ldap",
	97:  "client-machine-identifier",
	98:  "open-group-user-authentication",
	99:  "geo-conf-civic",
	100: "ieee-10031-tz-string",
	101: "reference-to-tz-database",
	112: "netinfo-parent-server-address",
	113: "netinfo-parent-server-tag",
	114: "url",
	116: "auto-configure",
	117: "name-service-search",
	118: "subnet-selection",
	119: "dns-domain-search-list",
	120: "sip-servers",
	121: "classless-static-route",
	122: "cablelabs-client-configuration",
	123: "geo-conf",
	124: "vendor-identifying-vendor-class",
	125: "vendor-identifying-vendor-specific",
	128: "tftp-server-ip-address",
	129: "call-server-ip-address",
	130: "discrimination-string",
	131: "remote-statistics-server-ip-address",
	132: "8021p-vlan-id",
	133: "8021q-l2-priority",
	134: "diffserv-code-point",
	135: "http-proxy-for-phone-specific-applications",
	136: "pana-authentication-agent",
	137: "lo-st-server",
	138: "cap-wap-access-controller-address",
	139: "option-ipv4-address-mos",
	140: "option-ipv4-fqdn-mos",
	141: "sip-ua-configuration-service-domains",
	142: "option-ipv4-address-andsf",
	143: "option-ipv6-address-andsf",
	150: "tftp-server-address",
	151: "status-code",
	152: "base-time",
	153: "start-time-of-state",
	154: "query-start-time",
	155: "query-end-time",
	156: "dhcp-state",
	157: "data-source",
	175: "etherboot",
	176: "ip-telephone",
	177: "etherboot-packet-cable-and-cablehome",
	208: "pxelinux-magic-string",
	209: "pxelinux-config-file",
	210: "pxelinux-path-prefix",
	211: "pxelinux-reboot-time",
	212: "option-6rd",
	213: "option-v4-access-domain",
	220: "subnet-allocation",
	221: "virtual-subnet-selection",
	255: "end",
}

func code4ToDescription(code uint8) string {
	if description, ok := code4Description[code]; ok {
		return description
	} else {
		return "unassigned"
	}
}
