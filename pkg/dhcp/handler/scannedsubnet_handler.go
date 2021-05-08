package handler

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	dhcpresource "github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

const (
	DefaultScanInterval   = 300 //5 min
	DefaultSearchInterval = 750 //12 min 30s
	IllegalDhcpAlarm      = "illegal_dhcp_alarm"
	ThresholdDhcpTopic    = "threshold_dhcp"
)

var (
// ThresholdIDIllegalDHCP = strings.ToLower(string(alarm.ThresholdNameIllegalDHCP))
)

type DHCPSubnet struct {
	subnet              *dhcpresource.Subnet
	pools               []*dhcpresource.Pool
	reservations        []*dhcpresource.Reservation
	staticAddresses     []*dhcpresource.StaticAddress
	capacity            uint64
	poolCapacity        uint64
	dynamicPoolCapacity uint64
	reservationRatio    string
	staticAddressRatio  string
	unusedRatio         string
}

type ScannedSubnetHandler struct {
	scannedSubnets map[string]*ScannedSubnetAndNICs
	dhcpClient     *dhcpclient.DHCPClient
	lock           sync.RWMutex
}

type ScannedSubnetAndNICs struct {
	ipnet net.IPNet
}

func NewScannedSubnetHandler() (*ScannedSubnetHandler, error) {
	searchInterval := DefaultSearchInterval
	if conf := config.GetConfig(); conf != nil {
		if conf.IllegalDHCPServerScan.Interval != 0 {
			searchInterval = int(conf.IllegalDHCPServerScan.Interval)
		}
	}

	dhcpClient, err := dhcpclient.New()
	if err != nil {
		return nil, err
	}

	h := &ScannedSubnetHandler{
		scannedSubnets: make(map[string]*ScannedSubnetAndNICs),
		dhcpClient:     dhcpClient,
	}

	go h.searchIllegalDHCPServer(searchInterval)
	return h, nil
}

func (h *ScannedSubnetHandler) searchIllegalDHCPServer(searchInterval int) {
	ticker := time.NewTicker(time.Duration(searchInterval) * time.Second)
	defer ticker.Stop()
	defer h.dhcpClient.Close()
	for {
		select {
		case <-ticker.C:
			getDHCPNodeList()

			dhcpServers := h.dhcpClient.FindIllegalDHCPServer()
			// var thresholds []*alarm.Threshold
			// if _, err := restdb.GetResourceWithID(db.GetDB(), ThresholdIDIllegalDHCP, &thresholds); err != nil {
			// 	log.Warnf("load threshold ipconflict failed: %s", err.Error())
			// 	continue
			// }

			sendEventWithIllegalDHCPIfNeed(dhcpServers)
		}
	}
}

func sendEventWithIllegalDHCPIfNeed(dhcpServers []*dhcpclient.DHCPServer) {
	if len(dhcpServers) == 0 {
		return
	}

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:   config.GetConfig().Kafka.Addr,
		Topic:     ThresholdDhcpTopic,
		BatchSize: 1,
		Dialer: &kafka.Dialer{
			Timeout:   time.Second * 10,
			DualStack: true,
			KeepAlive: time.Second * 5},
	})

	defer w.Close()

	for _, dhcpServer := range dhcpServers {
		ip := dhcpServer.IPv4
		if ip == "" {
			ip = dhcpServer.IPv6
		}

		err := w.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(IllegalDhcpAlarm),
				Value: []byte(ip),
			},
		)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func getKafkaConn() {
}
