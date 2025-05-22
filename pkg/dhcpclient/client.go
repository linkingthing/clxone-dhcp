package dhcpclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/linkingthing/cement/log"
	consulutil "github.com/linkingthing/clxone-utils/consul"
	"google.golang.org/grpc"

	"github.com/linkingthing/clxone-dhcp/config"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
)

var (
	LocalhostIPv4 = net.ParseIP("127.0.0.1")
	LocalhostIPv6 = net.ParseIP("::1")
)

const (
	DefaultWriteTimeout      = 10 * time.Second
	DefaultReadTimeout       = 10 * time.Second
	MaxUDPReceivedPacketSize = 8192
)

type Client interface {
	Close()
	Exchange() ([]*DHCPServer, error)
}

type DHCPServer struct {
	Mac  string
	IPv4 string
	IPv6 string
}

type DHCPClient struct {
	clients []Client
}

func New() (*DHCPClient, error) {
	clients, err := getClients()
	if err != nil {
		return nil, err
	}

	return &DHCPClient{clients: clients}, nil
}

func getDHCPNodeList() map[string]struct{} {
	nodes := make(map[string]struct{})
	if dhcpNodes, err := getDHCPNodesFromWarden(); err != nil {
		log.Warnf("get dhcp nodes from warden failed: %s", err.Error())
	} else {
		for _, dhcpNode := range dhcpNodes {
			nodes[dhcpNode.GetIpv4()] = struct{}{}
			nodes[dhcpNode.GetIpv6()] = struct{}{}
			nodes[dhcpNode.GetVirtualIp()] = struct{}{}
			nodes[dhcpNode.GetVirtualIpv6()] = struct{}{}
		}
	}

	if dhcpNodeIps, err := getDHCPNodesFromDHCPAgent(); err != nil {
		log.Warnf("get dhcp nodes from dhcp agent failed: %s", err.Error())
	} else {
		for _, dhcpNodeIp := range dhcpNodeIps {
			nodes[dhcpNodeIp] = struct{}{}
		}
	}

	return nodes
}

func getDHCPNodesFromDHCPAgent() ([]string, error) {
	endpoints, err := consulutil.GetEndpoints(config.ConsulConfig, config.GetConfig().Consul.CallServices.DhcpAgent)
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, end := range endpoints {
		response, err := end(context.Background(), struct{}{})
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, response.(string), grpc.WithBlock(), grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		client := pbdhcpagent.NewDHCPManagerClient(conn)
		resp, err := client.GetDHCPNodes(ctx, &pbdhcpagent.GetDHCPNodesRequest{})
		if err != nil {
			return nil, err
		}

		ips = append(ips, resp.GetIpv4S()...)
		ips = append(ips, resp.GetIpv6S()...)
	}

	return ips, nil
}

func getDHCPNodesFromWarden() ([]*pbmonitor.Node, error) {
	if dhcpNodes, err := transport.GetDHCPNodes(); err != nil {
		return nil, err
	} else {
		return dhcpNodes.GetNodes(), nil
	}
}

func (cli *DHCPClient) ScanIllegalDHCPServer() []*DHCPServer {
	dhcpServer4s := make(map[string]*DHCPServer)
	dhcpServer6s := make(map[string]*DHCPServer)
	for _, client := range cli.clients {
		if servers, err := client.Exchange(); err != nil {
			log.Infof("exchange message with dhcp server failed: %s", err.Error())
			continue
		} else {
			nodes := getDHCPNodeList()
			if len(nodes) == 0 {
				continue
			}

			for _, server := range servers {
				if server.IPv4 != "" {
					if _, ok := nodes[server.IPv4]; !ok {
						dhcpServer4s[server.IPv4] = server
					}
				} else {
					if _, ok := nodes[server.IPv6]; !ok {
						dhcpServer6s[server.IPv6] = server
					}
				}
			}
		}
	}

	dhcpServers := make([]*DHCPServer, 0, len(dhcpServer4s)+len(dhcpServer6s))
	for _, server := range dhcpServer4s {
		dhcpServers = append(dhcpServers, server)
	}

	for _, server := range dhcpServer6s {
		dhcpServers = append(dhcpServers, server)
	}

	return dhcpServers
}

func (cli *DHCPClient) Close() {
	for _, client := range cli.clients {
		client.Close()
	}
}

func getClients() ([]Client, error) {
	var clients []Client
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("get interfaces failed: %s", err.Error())
	}

	clientV4 := false
	clientV6Linklocal := false
	clientV6Global := false
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipnet.IP.Equal(LocalhostIPv4) || ipnet.IP.Equal(LocalhostIPv6) {
				break
			}

			if ip := ipnet.IP; ip.To4() != nil {
				if !clientV4 && ip.IsGlobalUnicast() {
					if client, err := newClient4(iface); err == nil {
						clientV4 = true
						clients = append(clients, client)
					}
				}
			} else {
				if !clientV6Linklocal && ip.IsLinkLocalUnicast() {
					if client, err := newClient6(iface, ip); err == nil {
						clientV6Linklocal = true
						clients = append(clients, client)
					}
				}

				if !clientV6Global && ip.IsGlobalUnicast() {
					if client, err := newClient6(iface, ip); err == nil {
						clientV6Global = true
						clients = append(clients, client)
					}
				}
			}
		}

		if clientV4 && clientV6Linklocal && clientV6Global {
			break
		}
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("no found valid interface for dhcp client")
	}

	return clients, nil
}
