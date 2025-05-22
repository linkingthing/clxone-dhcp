package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"sync"

	"github.com/golang/protobuf/proto"
	kg "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type DHCPCmd string

const (
	TopicPrefix = "dhcp_"

	CreateSubnet4sAndPools DHCPCmd = "create_subnet4s_and_pools"
	DeleteSubnet4s         DHCPCmd = "delete_subnet4s"

	CreateSharedNetwork4  DHCPCmd = "create_sharednetwork4"
	UpdateSharedNetwork4  DHCPCmd = "update_sharednetwork4"
	DeleteSharedNetwork4  DHCPCmd = "delete_sharednetwork4"
	CreateSharedNetwork4s DHCPCmd = "create_sharednetwork4s"
	DeleteSharedNetwork4s DHCPCmd = "delete_sharednetwork4s"

	CreateSubnet4 DHCPCmd = "create_subnet4"
	UpdateSubnet4 DHCPCmd = "update_subnet4"
	DeleteSubnet4 DHCPCmd = "delete_subnet4"

	CreatePool4  DHCPCmd = "create_pool4"
	DeletePool4  DHCPCmd = "delete_pool4"
	CreatePool4s DHCPCmd = "create_pool4s"
	DeletePool4s DHCPCmd = "delete_pool4s"

	CreateReservedPool4  DHCPCmd = "create_reservedpool4"
	DeleteReservedPool4  DHCPCmd = "delete_reservedpool4"
	CreateReservedPool4s DHCPCmd = "create_reservedpool4s"
	DeleteReservedPool4s DHCPCmd = "delete_reservedpool4s"

	CreateReservation4  DHCPCmd = "create_reservation4"
	DeleteReservation4  DHCPCmd = "delete_reservation4"
	CreateReservation4s DHCPCmd = "create_reservation4s"
	DeleteReservation4s DHCPCmd = "delete_reservation4s"

	CreateClientClass4  DHCPCmd = "create_clientclass4"
	DeleteClientClass4  DHCPCmd = "delete_clientclass4"
	UpdateClientClass4  DHCPCmd = "update_clientclass4"
	CreateClientClass4s DHCPCmd = "create_clientclass4s"
	DeleteClientClass4s DHCPCmd = "delete_clientclass4s"

	CreateSubnet6sAndPools DHCPCmd = "create_subnet6s_and_pools"
	DeleteSubnet6s         DHCPCmd = "delete_subnet6s"

	CreateSubnet6 DHCPCmd = "create_subnet6"
	UpdateSubnet6 DHCPCmd = "update_subnet6"
	DeleteSubnet6 DHCPCmd = "delete_subnet6"

	CreatePool6  DHCPCmd = "create_pool6"
	DeletePool6  DHCPCmd = "delete_pool6"
	CreatePool6s DHCPCmd = "create_pool6s"
	DeletePool6s DHCPCmd = "delete_pool6s"

	CreateReservedPool6  DHCPCmd = "create_reservedpool6"
	DeleteReservedPool6  DHCPCmd = "delete_reservedpool6"
	CreateReservedPool6s DHCPCmd = "create_reservedpool6s"
	DeleteReservedPool6s DHCPCmd = "delete_reservedpool6s"

	CreatePdPool  DHCPCmd = "create_pdpool"
	DeletePdPool  DHCPCmd = "delete_pdpool"
	CreatePdPools DHCPCmd = "create_pdpools"
	DeletePdPools DHCPCmd = "delete_pdpools"

	CreateReservedPdPool  DHCPCmd = "create_reservedpdpool"
	DeleteReservedPdPool  DHCPCmd = "delete_reservedpdpool"
	CreateReservedPdPools DHCPCmd = "create_reservedpdpools"
	DeleteReservedPdPools DHCPCmd = "delete_reservedpdpools"

	CreateReservation6  DHCPCmd = "create_reservation6"
	DeleteReservation6  DHCPCmd = "delete_reservation6"
	CreateReservation6s DHCPCmd = "create_reservation6s"
	DeleteReservation6s DHCPCmd = "delete_reservation6s"

	CreateClientClass6  DHCPCmd = "create_clientclass6"
	DeleteClientClass6  DHCPCmd = "delete_clientclass6"
	UpdateClientClass6  DHCPCmd = "update_clientclass6"
	CreateClientClass6s DHCPCmd = "create_clientclass6s"
	DeleteClientClass6s DHCPCmd = "delete_clientclass6s"

	CreateFingerprint  DHCPCmd = "create_fingerprint"
	DeleteFingerprint  DHCPCmd = "delete_fingerprint"
	UpdateFingerprint  DHCPCmd = "update_fingerprint"
	CreateFingerprints DHCPCmd = "create_fingerprints"
	DeleteFingerprints DHCPCmd = "delete_fingerprints"

	UpdatePinger DHCPCmd = "update_pinger"

	CreateOui  DHCPCmd = "create_oui"
	DeleteOui  DHCPCmd = "delete_oui"
	UpdateOui  DHCPCmd = "update_oui"
	CreateOuis DHCPCmd = "create_ouis"
	DeleteOuis DHCPCmd = "delete_ouis"

	UpdateAdmit             DHCPCmd = "update_admit"
	CreateAdmitMac          DHCPCmd = "create_admitmac"
	DeleteAdmitMac          DHCPCmd = "delete_admitmac"
	UpdateAdmitMac          DHCPCmd = "update_admitmac"
	CreateAdmitMacs         DHCPCmd = "create_admitmacs"
	DeleteAdmitMacs         DHCPCmd = "delete_admitmacs"
	CreateAdmitDuid         DHCPCmd = "create_admitduid"
	DeleteAdmitDuid         DHCPCmd = "delete_admitduid"
	UpdateAdmitDuid         DHCPCmd = "update_admitduid"
	CreateAdmitDuids        DHCPCmd = "create_admitduids"
	DeleteAdmitDuids        DHCPCmd = "delete_admitduids"
	CreateAdmitFingerprint  DHCPCmd = "create_admitfingerprint"
	DeleteAdmitFingerprint  DHCPCmd = "delete_admitfingerprint"
	UpdateAdmitFingerprint  DHCPCmd = "update_admitfingerprint"
	CreateAdmitFingerprints DHCPCmd = "create_admitfingerprints"
	DeleteAdmitFingerprints DHCPCmd = "delete_admitfingerprints"

	UpdateRateLimit      DHCPCmd = "update_ratelimit"
	CreateRateLimitMac   DHCPCmd = "create_ratelimitmac"
	DeleteRateLimitMac   DHCPCmd = "delete_ratelimitmac"
	UpdateRateLimitMac   DHCPCmd = "update_ratelimitmac"
	CreateRateLimitMacs  DHCPCmd = "create_ratelimitmacs"
	DeleteRateLimitMacs  DHCPCmd = "delete_ratelimitmacs"
	CreateRateLimitDuid  DHCPCmd = "create_ratelimitduid"
	DeleteRateLimitDuid  DHCPCmd = "delete_ratelimitduid"
	UpdateRateLimitDuid  DHCPCmd = "update_ratelimitduid"
	CreateRateLimitDuids DHCPCmd = "create_ratelimitduids"
	DeleteRateLimitDuids DHCPCmd = "delete_ratelimitduids"

	CreateAddressCode               DHCPCmd = "create_addresscode"
	DeleteAddressCode               DHCPCmd = "delete_addresscode"
	UpdateAddressCode               DHCPCmd = "update_addresscode"
	CreateAddressCodes              DHCPCmd = "create_addresscodes"
	DeleteAddressCodes              DHCPCmd = "delete_addresscodes"
	CreateAddressCodeLayout         DHCPCmd = "create_addresscode_layout"
	DeleteAddressCodeLayout         DHCPCmd = "delete_addresscode_layout"
	UpdateAddressCodeLayout         DHCPCmd = "update_addresscode_layout"
	CreateAddressCodeLayouts        DHCPCmd = "create_addresscode_layouts"
	DeleteAddressCodeLayouts        DHCPCmd = "delete_addresscode_layouts"
	CreateAddressCodeLayoutSegment  DHCPCmd = "create_addresscode_layout_segment"
	DeleteAddressCodeLayoutSegment  DHCPCmd = "delete_addresscode_layout_segment"
	UpdateAddressCodeLayoutSegment  DHCPCmd = "update_addresscode_layout_segment"
	CreateAddressCodeLayoutSegments DHCPCmd = "create_addresscode_layout_segments"
	DeleteAddressCodeLayoutSegments DHCPCmd = "delete_addresscode_layout_segments"

	CreateAsset  DHCPCmd = "create_asset"
	DeleteAsset  DHCPCmd = "delete_asset"
	UpdateAsset  DHCPCmd = "update_asset"
	CreateAssets DHCPCmd = "create_assets"
	DeleteAssets DHCPCmd = "delete_assets"
)

var globalDHCPAgentService *DHCPAgentService

var onceDHCPAgentService sync.Once

type DHCPAgentService struct {
	dhcpWriter    *kg.Writer
	NodeCache     map[string]*pbmonitor.Node
	HostnameCache map[string]*pbmonitor.Node
}

func GetDHCPAgentService() *DHCPAgentService {
	onceDHCPAgentService.Do(func() {
		globalDHCPAgentService = &DHCPAgentService{
			dhcpWriter: &kg.Writer{
				Transport: &kg.Transport{
					SASL: plain.Mechanism{
						Username: config.GetConfig().Kafka.Username,
						Password: config.GetConfig().Kafka.Password,
					},
					TLS: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
				Addr:       kg.TCP(config.GetConfig().Kafka.Addrs...),
				BatchSize:  1,
				BatchBytes: 10e8,
				Balancer:   &kg.LeastBytes{},
			},
		}
	})
	return globalDHCPAgentService
}

func (a *DHCPAgentService) SendDHCPCmdWithNodes(nodes []string, cmd DHCPCmd, msg proto.Message) ([]string, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, errorno.ErrHandleCmd(string(cmd), err.Error())
	}

	succeedNodes := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if cache, ok := a.NodeCache[node]; ok {
			node = cache.GetHostname()
		}

		if err := a.dhcpWriter.WriteMessages(context.Background(), kg.Message{
			Topic: TopicPrefix + node, Key: []byte(cmd), Value: data}); err != nil {
			return succeedNodes, errorno.ErrHandleCmd(string(cmd), err.Error())
		}

		succeedNodes = append(succeedNodes, node)
	}

	return succeedNodes, nil
}

func (a *DHCPAgentService) InitNodeCache() error {
	dhcpNodes, err := transport.GetDHCPNodes()
	if err != nil {
		return fmt.Errorf("init dhcp nodes cache failed:%s", err.Error())
	}

	a.NodeCache = make(map[string]*pbmonitor.Node, len(dhcpNodes.GetNodes()))
	a.HostnameCache = make(map[string]*pbmonitor.Node, len(dhcpNodes.GetNodes()))
	for _, node := range dhcpNodes.GetNodes() {
		a.NodeCache[node.GetIpv4()] = node
		a.HostnameCache[node.GetHostname()] = node
	}
	return nil
}
