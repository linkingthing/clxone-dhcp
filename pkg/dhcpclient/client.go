package dhcpclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/pb"
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

func getDHCPNodeList() (nodes []resource.Node, err error) {
	endpoints, err := pb.GetEndpoints("clxone-dhcp-agent")
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	for _, end := range endpoints {
		response, err := end(context.Background(), struct{}{})
		if err != nil {
			logrus.Error(err)
			return nil, err
		}

		logrus.Debug(response)
		// TODO: rpc all agent to get all nodes
		nodes = append(nodes, resource.Node{})

	}

	return
}

func (cli *DHCPClient) FindIllegalDHCPServer() []*DHCPServer {
	dhcpServer4s := make(map[string]*DHCPServer)
	dhcpServer6s := make(map[string]*DHCPServer)
	for _, client := range cli.clients {
		if servers, err := client.Exchange(); err != nil {
			log.Debugf("exchange message with dhcp server failed: %s", err.Error())
			continue
		} else {
			nodes, err := getDHCPNodeList()
			if err != nil {
				log.Warnf("get dhcp node from db failed: %s", err.Error())
			}

			for _, server := range servers {
				if server.IPv4 != "" {
					if isDHCPNodeIPv4(nodes, server.IPv4) == false {
						dhcpServer4s[server.IPv4] = server
					}
				} else {
					if isDHCPNodeIPv6(nodes, server.IPv6) == false {
						dhcpServer6s[server.IPv6] = server
					}
				}
			}
		}
	}

	var dhcpServers []*DHCPServer
	for _, server := range dhcpServer4s {
		dhcpServers = append(dhcpServers, server)
	}

	for _, server := range dhcpServer6s {
		dhcpServers = append(dhcpServers, server)
	}

	return dhcpServers
}

func isDHCPNodeIPv4(nodes []resource.Node, ip string) bool {
	for _, node := range nodes {
		if node.Ip == ip {
			return true
		}
	}

	return false
}

func isDHCPNodeIPv6(nodes []resource.Node, ip string) bool {
	for _, node := range nodes {
		if slice.SliceIndex(node.Ipv6s, ip) != -1 {
			return true
		}
	}

	return false
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
		return nil, err
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
			if ok == false {
				continue
			}

			if ip := ipnet.IP; ip.To4() != nil {
				if clientV4 == false && ip.IsGlobalUnicast() {
					if client, err := newClient4(iface); err == nil {
						clientV4 = true
						clients = append(clients, client)
					}
				}
			} else {
				if clientV6Linklocal == false && ip.IsLinkLocalUnicast() {
					if client, err := newClient6(iface, ip); err == nil {
						clientV6Linklocal = true
						clients = append(clients, client)
					}
				}

				if clientV6Global == false && ip.IsGlobalUnicast() {
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
