package api

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
	"github.com/linkingthing/clxone-dhcp/pkg/proto/alarm"
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

	dhcpClient, err := dhcpclient.New()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	h := &ScannedSubnetHandler{
		dhcpClient: dhcpClient,
	}

	alarmService := service.NewAlarmService()
	err = alarmService.RegisterThresholdToKafka(service.RegisterThreshold, alarmService.DhcpThreshold)
	if err != nil {
		panic(err)
	}

	go alarmService.HandleUpdateThresholdEvent(service.ThresholdDhcpTopic, alarmService.UpdateDhcpThresHold)

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
			if alarmService.DhcpThreshold.Enabled {
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
}
