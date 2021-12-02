package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	kg "github.com/segmentio/kafka-go"

	"github.com/linkingthing/clxone-dhcp/config"
)

type DHCPCmd string

const (
	TopicDHCP   = "dhcp"
	TopicPrefix = "dhcp_"

	CreateSubnet4sAndPools DHCPCmd = "create_subnet4s_and_pools"
	DeleteSubnet4s         DHCPCmd = "delete_subnet4s"

	CreateSharedNetwork4 DHCPCmd = "create_sharednetwork4"
	UpdateSharedNetwork4 DHCPCmd = "update_sharednetwork4"
	DeleteSharedNetwork4 DHCPCmd = "delete_sharednetwork4"

	CreateSubnet4 DHCPCmd = "create_subnet4"
	UpdateSubnet4 DHCPCmd = "update_subnet4"
	DeleteSubnet4 DHCPCmd = "delete_subnet4"

	CreatePool4 DHCPCmd = "create_pool4"
	DeletePool4 DHCPCmd = "delete_pool4"

	CreateReservedPool4 DHCPCmd = "create_reservedpool4"
	DeleteReservedPool4 DHCPCmd = "delete_reservedpool4"

	CreateReservation4 DHCPCmd = "create_reservation4"
	DeleteReservation4 DHCPCmd = "delete_reservation4"

	CreateClientClass4 DHCPCmd = "create_clientclass4"
	DeleteClientClass4 DHCPCmd = "delete_clientclass4"
	UpdateClientClass4 DHCPCmd = "update_clientclass4"

	CreateSubnet6sAndPools DHCPCmd = "create_subnet6s_and_pools"
	DeleteSubnet6s         DHCPCmd = "delete_subnet6s"

	CreateSubnet6 DHCPCmd = "create_subnet6"
	UpdateSubnet6 DHCPCmd = "update_subnet6"
	DeleteSubnet6 DHCPCmd = "delete_subnet6"

	CreatePool6 DHCPCmd = "create_pool6"
	DeletePool6 DHCPCmd = "delete_pool6"

	CreateReservedPool6 DHCPCmd = "create_reservedpool6"
	DeleteReservedPool6 DHCPCmd = "delete_reservedpool6"

	CreatePdPool DHCPCmd = "create_pdpool"
	DeletePdPool DHCPCmd = "delete_pdpool"

	CreateReservedPdPool DHCPCmd = "create_reservedpdpool"
	DeleteReservedPdPool DHCPCmd = "delete_reservedpdpool"

	CreateReservation6 DHCPCmd = "create_reservation6"
	DeleteReservation6 DHCPCmd = "delete_reservation6"

	CreateClientClass6 DHCPCmd = "create_clientclass6"
	DeleteClientClass6 DHCPCmd = "delete_clientclass6"
	UpdateClientClass6 DHCPCmd = "update_clientclass6"

	CreateFingerprint DHCPCmd = "create_fingerprint"
	DeleteFingerprint DHCPCmd = "delete_fingerprint"
	UpdateFingerprint DHCPCmd = "update_fingerprint"

	UpdatePinger DHCPCmd = "update_pinger"
)

var globalDHCPAgentService *DHCPAgentService
var onceDHCPAgentService sync.Once

type DHCPAgentService struct {
	dhcpWriter *kg.Writer
}

func GetDHCPAgentService() *DHCPAgentService {
	onceDHCPAgentService.Do(func() {
		globalDHCPAgentService = &DHCPAgentService{
			dhcpWriter: kg.NewWriter(kg.WriterConfig{
				Brokers:    config.GetConfig().Kafka.Addrs,
				BatchSize:  1,
				BatchBytes: 10e8,
				Dialer: &kg.Dialer{
					Timeout:   time.Second * 10,
					DualStack: true,
					KeepAlive: time.Second * 5},
			})}
	})
	return globalDHCPAgentService
}

func (a *DHCPAgentService) SendDHCPCmdWithNodes(nodes []string, cmd DHCPCmd, msg proto.Message) ([]string, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal %s failed: %s\n", cmd, err.Error())
	}

	var succeedNodes []string
	for _, node := range nodes {
		if err := a.dhcpWriter.WriteMessages(context.Background(), kg.Message{
			Topic: TopicPrefix + node, Key: []byte(cmd), Value: data}); err != nil {
			return succeedNodes, fmt.Errorf("send kafka cmd to %s failed: %s", node, err.Error())
		}

		succeedNodes = append(succeedNodes, node)
	}

	return succeedNodes, nil
}

func (a *DHCPAgentService) SendDHCPCmd(cmd DHCPCmd, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %s failed: %s\n", cmd, err.Error())
	}

	return a.dhcpWriter.WriteMessages(context.Background(), kg.Message{
		Topic: TopicDHCP, Key: []byte(cmd), Value: data})
}
