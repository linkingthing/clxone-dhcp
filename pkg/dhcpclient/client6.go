package dhcpclient

import (
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv6"
)

var AllDHCPv6ServerAddr = &net.UDPAddr{IP: dhcpv6.AllDHCPRelayAgentsAndServers, Port: dhcpv6.DefaultServerPort}

type Client6 struct {
	solicit *dhcpv6.Message
	conn    *net.UDPConn
}

func newClient6(iface net.Interface, localIP net.IP) (Client, error) {
	solicit, err := dhcpv6.NewSolicit(iface.HardwareAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp6", &net.UDPAddr{IP: localIP, Port: dhcpv6.DefaultClientPort, Zone: iface.Name})
	if err != nil {
		return nil, err
	}

	return &Client6{
		solicit: solicit,
		conn:    conn,
	}, nil
}

func (cli *Client6) Close() {
	cli.conn.Close()
}

func (cli *Client6) Exchange() ([]*DHCPServer, error) {
	packet := cli.solicit
	transId, err := dhcpv6.GenerateTransactionID()
	if err != nil {
		return nil, err
	}

	packet.TransactionID = transId
	if err := cli.conn.SetWriteDeadline(time.Now().Add(DefaultReadTimeout)); err != nil {
		return nil, err
	}

	_, err = cli.conn.WriteTo(packet.ToBytes(), AllDHCPv6ServerAddr)
	if err != nil {
		return nil, err
	}

	oobdata := []byte{}
	if err := cli.conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout)); err != nil {
		return nil, err
	}

	var dhcpServers []*DHCPServer
	for {
		buf := make([]byte, MaxUDPReceivedPacketSize)
		n, _, _, addr, err := cli.conn.ReadMsgUDP(buf, oobdata)
		if err != nil {
			break
		}

		advertise, err := dhcpv6.FromBytes(buf[:n])
		if err != nil {
			continue
		}

		if recvMsg, ok := advertise.(*dhcpv6.Message); ok {
			if packet.TransactionID != recvMsg.TransactionID {
				continue
			}
		}

		var mac string
		if serverMac, err := getServerMac(advertise); err == nil {
			mac = serverMac.String()
		}

		dhcpServers = append(dhcpServers, &DHCPServer{
			Mac:  mac,
			IPv6: addr.IP.String(),
		})
	}

	return dhcpServers, nil
}

func getServerMac(packet dhcpv6.DHCPv6) (net.HardwareAddr, error) {
	var msg *dhcpv6.Message
	if packet.IsRelay() {
		if innerMsg, err := packet.(*dhcpv6.RelayMessage).GetInnerMessage(); err != nil {
			return nil, err
		} else {
			msg = innerMsg
		}
	} else {
		msg = packet.(*dhcpv6.Message)
	}

	if serverId := msg.Options.ServerID(); serverId != nil && serverId.LinkLayerAddr != nil {
		return serverId.LinkLayerAddr, nil
	}

	return nil, fmt.Errorf("no found server mac")
}
