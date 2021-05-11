package services

import (
	"context"
	"sync"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	kg "github.com/segmentio/kafka-go"
)

const (
	Topic = "dhcp"

	CreateSubnet4 = "create_subnet4"
	UpdateSubnet4 = "update_subnet4"
	DeleteSubnet4 = "delete_subnet4"

	CreateSubnet6 = "create_subnet6"
	UpdateSubnet6 = "update_subnet6"
	DeleteSubnet6 = "delete_subnet6"

	CreatePool4 = "create_pool4"
	DeletePool4 = "delete_pool4"
	UpdatePool4 = "update_pool4"

	CreatePool6 = "create_pool6"
	DeletePool6 = "delete_pool6"
	UpdatePool6 = "update_pool6"

	CreatePDPool = "create_pdpool"
	DeletePDPool = "delete_pdpool"
	UpdatePDPool = "update_pdpool"

	CreateReservation4 = "create_reservation4"
	DeleteReservation4 = "delete_reservation4"
	UpdateReservation4 = "update_reservation4"

	CreateReservation6 = "create_reservation6"
	DeleteReservation6 = "delete_reservation6"
	UpdateReservation6 = "update_reservation6"

	CreateClientClass4 = "create_clientclass4"
	DeleteClientClass4 = "delete_clientclass4"
	UpdateClientClass4 = "update_clientclass4"

	UpdateGlobalConfig = "update_globalconfig"
)

var globalDHCPAgentService *DHCPAgentService
var onceDHCPAgentService sync.Once

type DHCPAgentService struct {
	dhcpWriter *kg.Writer
}

func NewDHCPAgentService() *DHCPAgentService {
	onceDHCPAgentService.Do(func() {
		globalDHCPAgentService = &DHCPAgentService{}

		globalDHCPAgentService.dhcpWriter = kg.NewWriter(kg.WriterConfig{
			Brokers:   config.GetConfig().Kafka.Addr,
			Topic:     Topic,
			BatchSize: 1,
			Dialer: &kg.Dialer{
				Timeout:   time.Second * 10,
				DualStack: true,
				KeepAlive: time.Second * 5},
		})
	})
	return globalDHCPAgentService
}

func (a *DHCPAgentService) SendDHCPCmd(cmd string, data []byte) error {
	return a.dhcpWriter.WriteMessages(context.Background(), kg.Message{Key: []byte(cmd), Value: data})
}
