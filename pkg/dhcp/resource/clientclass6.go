package resource

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableClientClass6 = restdb.ResourceDBType(&ClientClass6{})

type ClientClass6 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string          `json:"name" rest:"required=true,description=immutable" db:"uk"`
	Code                      uint16          `json:"code" rest:"required=true,description=immutable"`
	Condition                 OptionCondition `json:"condition" rest:"required=true,options=exists|equal|substring"`
	Regexp                    string          `json:"regexp"`
	BeginIndex                uint32          `json:"beginIndex"`
	Description               string          `json:"description"`
}

func (c *ClientClass6) Validate() error {
	if len(c.Name) == 0 || (c.Condition != OptionConditionExists && len(c.Regexp) == 0) {
		return ErrNameOrRegexpMissing
	} else if c.Code < 1 || c.Code > 65535 {
		return fmt.Errorf("code %d not in [1, 65535]", c.Code)
	} else if err := util.ValidateStrings(c.Name, c.Regexp, c.Description); err != nil {
		return fmt.Errorf("name %s or regexp %s is invalid", c.Name, c.Regexp)
	} else {
		if c.Description == "" {
			c.Description = code6ToDescription(c.Code)
		}

		return nil
	}
}

var code6Description = map[uint16]string{
	1:   "client-identifier",
	2:   "server-identifier",
	3:   "iana",
	4:   "iata",
	5:   "ia-ip-address",
	6:   "requested-options",
	7:   "preference",
	8:   "elapsed-time",
	9:   "relay-message",
	11:  "auth",
	12:  "unicast",
	13:  "status-code",
	14:  "rapid-commit",
	15:  "user-class",
	16:  "vendor-class",
	17:  "vendor-opts",
	18:  "interface-id",
	19:  "reconfig-message",
	20:  "reconfig-accept",
	21:  "sip-servers-domain-name-list",
	22:  "sip-servers-ipv6-address-list",
	23:  "dns-recursive-name-server",
	24:  "domain-search-list",
	25:  "iapd",
	26:  "ia-prefix",
	27:  "nis-servers",
	28:  "nisp-servers",
	29:  "nis-domain-name",
	30:  "nisp-domain-name",
	31:  "sntp-server-list",
	32:  "information-refresh-time",
	33:  "bcmcs-controller-domain-name-list",
	34:  "bcmcs-controller-ipv6-address-list",
	36:  "geo-conf-civic",
	37:  "remote-id",
	38:  "relay-agent-subscriber-id",
	39:  "fqdn",
	40:  "pana-authentication-agent",
	41:  "new-posix-timezone",
	42:  "new-tzdb-timezone",
	43:  "echo-request",
	44:  "lq-query",
	45:  "client-data",
	46:  "clt-time",
	47:  "lq-relay-data",
	48:  "lq-client-link",
	49:  "mipv6-home-network-id-fqdn",
	50:  "mipv6-visited-home-network-information",
	51:  "lo-st-server",
	52:  "cap-wap-access-controller-address",
	53:  "relay-id",
	54:  "ipv6-address-mos",
	55:  "ipv6-fqdn-mos",
	56:  "ntp-server",
	57:  "v6-access-domain",
	58:  "sip-ua-cs-list",
	59:  "boot-file-url",
	60:  "boot-file-parameters",
	61:  "client-arch-type",
	62:  "network-interface-id",
	63:  "geo-location",
	64:  "aftr-name",
	65:  "erp-local-domain-name",
	66:  "rsoo",
	67:  "pd-exclude",
	68:  "virtual-subnet-selection",
	69:  "mipv6-identified-home-network-information",
	70:  "mipv6-unrestricted-home-network-information",
	71:  "mipv6-home-network-prefix",
	72:  "mipv6-home-agent-address",
	73:  "mipv6-home-agent-fqdn",
	74:  "rdnss-selection",
	75:  "kerberos-principal-name",
	76:  "kerberos-realm-name",
	77:  "kerberos-default-realm-name",
	78:  "kerberos-kdc",
	79:  "client-link-layer-address",
	80:  "link-address",
	81:  "radius",
	82:  "max-solicit-timeout-value",
	83:  "max-information-request-timeout-value",
	84:  "address-selection",
	85:  "address-selection-policy-table",
	86:  "port-controller-protocol-server",
	87:  "encapsulated-dhcpv4-message",
	88:  "dhcpv4-over-dhcpv6-server",
	89:  "softwire46-rule",
	90:  "softwire46-border-relay",
	91:  "softwire46-default-mapping-rule",
	92:  "softwire46-ipv4-ipv6-address-binding",
	93:  "softwire46-port-parameters",
	94:  "softwire46-map-e-container",
	95:  "softwire46-map-t-container",
	96:  "softwire46-light-weight-4over6-container",
	97:  "ipv4-residual-deployment",
	98:  "ipv4-residual-deployment-mapping-rule",
	99:  "ipv4-residual-deployment-non-mapping-rule",
	100: "leasequery-server-base-time",
	101: "leasequery-server-query-start-time",
	102: "leasequery-server-query-end-time",
	103: "captive-portal-uri",
	104: "mpl-parameters",
	105: "access-network-information-access-technology-type",
	106: "access-network-information-network-name",
	107: "access-network-information-access-point-name",
	108: "access-network-information-access-point-bssid",
	109: "access-network-information-operator-identifier",
	110: "access-network-information-operator-realm",
	111: "softwire46-priority",
	112: "manufacturer-usage-description-url",
	113: "v6-prefix64",
	114: "failover-binding-status",
	115: "failover-connection-flags",
	116: "failover-dns-removal-info",
	117: "failover-dns-removal-hostname",
	118: "failover-dns-removal-zone-name",
	119: "failover-dns-removal-flags",
	120: "failover-maximum-expiration-time",
	121: "failover-maximum-unacked-bndupd-messages",
	122: "failover-maximum-client-lead-time",
	123: "failover-partner-lifetime",
	124: "failover-received-partner-lifetime",
	125: "failover-last-partner-down-time",
	126: "failover-last-client-time",
	127: "failover-protocol-version",
	128: "failover-keepalive-time",
	129: "failover-reconfigure-data",
	130: "failover-relationship-name",
	131: "failover-server-flags",
	132: "failover-server-state",
	133: "failover-state-start-time",
	134: "failover-state-expiration-time",
	135: "relay-source-port",
	136: "ipv6-secure-zerotouch-provisioning-redirect",
	137: "softwire46-source-binding-prefix-hint",
	143: "ipv6-access-network-discovery-and-selection-function-address",
}

func code6ToDescription(code uint16) string {
	if description, ok := code6Description[code]; ok {
		return description
	} else {
		return "unassigned"
	}
}
