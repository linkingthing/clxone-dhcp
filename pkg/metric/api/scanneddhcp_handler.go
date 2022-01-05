package api

import (
	"time"

	"github.com/linkingthing/cement/log"
	pbutil "github.com/linkingthing/clxone-utils/alarm/proto"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
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
			threshold := GetAlarmService().GetThreshold(pbutil.ThresholdName_illegalDhcp)
			if threshold == nil {
				continue
			}
			dhcpServers := h.dhcpClient.ScanIllegalDHCPServer()
			for _, dhcpServer := range dhcpServers {
				ip := dhcpServer.IPv4
				if ip == "" {
					ip = dhcpServer.IPv6
				}
				if err := GetAlarmService().AddIllegalDHCPAlarm(ip, dhcpServer.Mac); err != nil {
					log.Warnf("add dhcp illegal alarm failed:%s", err.Error())
					continue
				}
			}
		}
	}
}
