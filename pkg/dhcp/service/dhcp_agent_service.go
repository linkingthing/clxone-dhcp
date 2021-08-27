package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/clxone-dhcp/config"
	kg "github.com/segmentio/kafka-go"
)

type DHCPCmd string

const (
	Topic = "dhcp"

	CreateSubnet4sAndPools DHCPCmd = "create_subnet4s_and_pools"

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
				Topic:      Topic,
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

func (a *DHCPAgentService) SendDHCPCmd(cmd DHCPCmd, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %s failed: %s\n", cmd, err.Error())
	}

	return a.dhcpWriter.WriteMessages(context.Background(), kg.Message{Key: []byte(cmd), Value: data})
}
