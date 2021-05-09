package handler

import (
	"sync"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/alarm"
	"github.com/sirupsen/logrus"
)

const (
	DefaultSearchInterval = 750 //12 min 30s
)

type ScannedSubnetHandler struct {
	dhcpClient *dhcpclient.DHCPClient
	lock       sync.RWMutex
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

	alarmService := service.NewAlarmService()
	err = alarmService.RegisterThresholdToKafka(service.IllegalDhcpAlarm, alarmService.DhcpThreshold)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	go alarmService.ListenUpdateThresholdEvent(service.UpdateThreshold, alarmService.UpdateDhcpThresHold)

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

			dhcpServers := h.dhcpClient.FindIllegalDHCPServer()
			alarmService := service.NewAlarmService()

			for _, dhcpServer := range dhcpServers {
				ip := dhcpServer.IPv4
				if ip == "" {
					ip = dhcpServer.IPv6
				}

				alarmService.SendEventWithValues(&alarm.IllegalDhcpAlarm{
					BaseAlarm: &alarm.BaseAlarm{
						BaseThreshold: alarmService.DhcpThreshold.BaseThreshold,
						Time:          time.Now().Format(time.RFC3339),
						SendMail:      alarmService.DhcpThreshold.SendMail,
					},
					IllegalDhcpIp:  ip,
					IllegalDhcpMac: dhcpServer.Mac,
				})
			}

		}
	}
}
