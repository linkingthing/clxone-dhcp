package api

import (
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
	pbalarm "github.com/linkingthing/clxone-dhcp/pkg/proto/alarm"
)

const (
	DefaultSearchInterval uint32 = 750 //12 min 30s
)

type ScannedDHCPHandler struct {
	dhcpClient *dhcpclient.DHCPClient
}

func InitScannedDHCPHandler(conf *config.DHCPConfig) error {
	searchInterval := DefaultSearchInterval
	if conf.DHCPScan.Interval != 0 {
		searchInterval = conf.DHCPScan.Interval
	}

	dhcpClient, err := dhcpclient.New()
	if err != nil {
		return err
	}

	alarmService := service.NewAlarmService()
	if err := alarmService.RegisterThresholdToKafka(service.RegisterThreshold,
		alarmService.DhcpThreshold); err != nil {
		return err
	}

	go alarmService.HandleUpdateThresholdEvent(service.ThresholdDhcpTopic, alarmService.UpdateDhcpThresHold)
	h := &ScannedDHCPHandler{dhcpClient: dhcpClient}
	go h.scanIllegalDHCPServer(searchInterval)

	return nil
}

func (h *ScannedDHCPHandler) scanIllegalDHCPServer(searchInterval uint32) {
	ticker := time.NewTicker(time.Duration(searchInterval) * time.Second)
	defer ticker.Stop()
	defer h.dhcpClient.Close()
	for {
		select {
		case <-ticker.C:
			if alarmService := service.NewAlarmService(); alarmService.DhcpThreshold.Enabled {
				dhcpServers := h.dhcpClient.ScanIllegalDHCPServer()
				for _, dhcpServer := range dhcpServers {
					ip := dhcpServer.IPv4
					if ip == "" {
						ip = dhcpServer.IPv6
					}

					alarmService.SendEventWithValues(service.AlarmKeyIllegalDhcp, &pbalarm.IllegalDhcpAlarm{
						BaseAlarm: &pbalarm.BaseAlarm{
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
