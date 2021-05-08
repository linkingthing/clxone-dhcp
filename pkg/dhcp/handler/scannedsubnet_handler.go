package handler

import (
	"net"
	"sync"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/alarm"
	"github.com/sirupsen/logrus"
)

const (
	DefaultScanInterval   = 300 //5 min
	DefaultSearchInterval = 750 //12 min 30s
)

// type DHCPSubnet struct {
// 	subnet              *dhcpresource.Subnet
// 	pools               []*dhcpresource.Pool
// 	reservations        []*dhcpresource.Reservation
// 	staticAddresses     []*dhcpresource.StaticAddress
// 	capacity            uint64
// 	poolCapacity        uint64
// 	dynamicPoolCapacity uint64
// 	reservationRatio    string
// 	staticAddressRatio  string
// 	unusedRatio         string
// }

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
		logrus.Error(err)
		return nil, err
	}

	h := &ScannedSubnetHandler{
		dhcpClient: dhcpClient,
	}

	alarmServie := service.NewAlarmService()
	err = alarmServie.RegisterThresholdToKafka(service.IllegalDhcpAlarm, alarmServie.DhcpThreshold)
	if err != nil {
		logrus.Error(err)
		return nil, err
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

			dhcpService := h.dhcpClient.FindIllegalDHCPServer()

			alarmService := service.NewAlarmService()
			alarmService.SendEventWithIllegalDHCP(dhcpService, alarm.IllegalDhcpAlarm{
				BaseAlarm: &alarm.BaseAlarm{
					BaseThreshold: alarmService.DhcpThreshold.BaseThreshold,
					Time:          time.Now().Format(time.RFC3339),
					SendMail:      alarmService.DhcpThreshold.SendMail,
				},
			})

		}
	}
}
