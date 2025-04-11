package service

import (
	"context"
	"time"

	nmap "github.com/Ullaakut/nmap/v2"
	"github.com/linkingthing/cement/log"
	pbutil "github.com/linkingthing/clxone-utils/alarm/proto"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type ScannedDHCPService struct {
	dhcpClient *dhcpclient.DHCPClient
	localIp    string
}

func InitScannedDHCPService(conf *config.DHCPConfig) error {
	dhcpClient, err := dhcpclient.New()
	if err != nil {
		return err
	}

	h := &ScannedDHCPService{dhcpClient: dhcpClient, localIp: conf.Server.IP}
	go h.scanIllegalDHCPServer(conf.DHCP.ScanInterval)

	return nil
}

func (h *ScannedDHCPService) scanIllegalDHCPServer(searchInterval uint32) {
	ticker := time.NewTicker(time.Duration(searchInterval) * time.Second)
	defer ticker.Stop()
	defer h.dhcpClient.Close()
	for {
		select {
		case <-ticker.C:
			if response, err := transport.IsNodeMaster(h.localIp); err == nil && !response.GetIsMaster() {
				continue
			}

			threshold := service.GetAlarmService().GetThreshold(pbutil.ThresholdName_illegalDhcp)
			if threshold == nil {
				continue
			}
			dhcpServers := h.dhcpClient.ScanIllegalDHCPServer()
			if err := fillIllegalDhcpServerMac(dhcpServers); err != nil {
				log.Warnf("fill illegal dhcp servers mac failed:%s", err.Error())
			}
			for _, dhcpServer := range dhcpServers {
				dhcpServer.Mac, _ = util.NormalizeMac(dhcpServer.Mac)
				ip := dhcpServer.IPv4
				if ip == "" {
					ip = dhcpServer.IPv6
				}
				if err := service.GetAlarmService().AddIllegalDHCPAlarm(ip, dhcpServer.Mac); err != nil {
					log.Warnf("add dhcp illegal alarm failed:%s", err.Error())
					continue
				}
			}
		}
	}
}

func fillIllegalDhcpServerMac(dhcpServers []*dhcpclient.DHCPServer) error {
	var scanIpv4s []string
	for _, server := range dhcpServers {
		if server.IPv4 != "" {
			scanIpv4s = append(scanIpv4s, server.IPv4)
		}
	}

	ipMacMap, err := nmapScanIpv4(scanIpv4s)
	if err != nil {
		return err
	}
	if len(ipMacMap) == 0 {
		return nil
	}

	for _, server := range dhcpServers {
		if mac, ok := ipMacMap[server.IPv4]; ok {
			server.Mac = mac
		}
	}

	return nil
}

func nmapScanIpv4(scanIps []string) (map[string]string, error) {
	if len(scanIps) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	scanner, err := nmap.NewScanner(
		nmap.WithTargets(scanIps...),
		nmap.WithPingScan(),
		nmap.WithDisabledDNSResolution(),
		nmap.WithHostTimeout(time.Second*5),
		nmap.WithContext(ctx),
	)
	if err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrMethodCreate, "nmap", err.Error())
	}

	result, warnings, err := scanner.Run()
	if err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrMethodAction, "nmap", err.Error())
	}
	if warnings != nil {
		log.Warnf("network scan warnings:%v\n", warnings)
	}

	ipMacMap := make(map[string]string)
	for _, host := range result.Hosts {
		if host.Status.String() == "up" {
			var ip, mac string
			for _, address := range host.Addresses {
				if address.AddrType == "ipv4" {
					ip = address.Addr
				} else if address.AddrType == "mac" {
					mac = address.Addr
				}
			}
			if mac != "" {
				ipMacMap[ip] = mac
			}
		}
	}

	return ipMacMap, nil
}
