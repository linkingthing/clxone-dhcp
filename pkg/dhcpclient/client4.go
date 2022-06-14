package dhcpclient

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/linkingthing/cement/log"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

type Client4 struct {
	discover         *dhcpv4.DHCPv4
	remoteSocketAddr unix.SockaddrInet4
	iface            net.Interface
}

func newClient4(iface net.Interface) (Client, error) {
	discover, err := dhcpv4.NewDiscovery(iface.HardwareAddr)
	if err != nil {
		return nil, err
	}

	var destination [net.IPv4len]byte
	copy(destination[:], net.IPv4bcast.To4())
	remoteSocketAddr := unix.SockaddrInet4{Port: dhcpv4.ClientPort, Addr: destination}

	return &Client4{
		discover:         discover,
		remoteSocketAddr: remoteSocketAddr,
		iface:            iface,
	}, nil
}

func (cli *Client4) Close() {}

func (cli *Client4) Exchange() ([]*DHCPServer, error) {
	sendfd, err := makeBroadcastSocket(cli.iface.Name)
	if err != nil {
		return nil, fmt.Errorf("make broadcast socket failed: %s", err.Error())
	}

	defer func() {
		if err := unix.Close(sendfd); err != nil {
			log.Debugf("unix close sendfd failed: %v", err.Error())
		}
	}()

	recvfd, err := makeListeningSocketWithCustomPort(cli.iface.Index)
	if err != nil {
		return nil, fmt.Errorf("make listening socket with custom port failed: %s", err.Error())
	}

	defer func() {
		if err := unix.Close(recvfd); err != nil {
			log.Debugf("unix close recvfd failed: %v", err.Error())
		}
	}()

	packet := cli.discover
	transId, err := dhcpv4.GenerateTransactionID()
	if err != nil {
		return nil, fmt.Errorf("gen transaction id failed: %s", err.Error())
	}

	packet.TransactionID = transId
	packetBytes, err := makeRawUDPPacket(packet.ToBytes())
	if err != nil {
		return nil, fmt.Errorf("make raw udp packet failed: %s", err.Error())
	}

	var dhcpServers []*DHCPServer
	errCh := make(chan error, 1)
	go func(errch chan<- error) {
		timeout := unix.NsecToTimeval(DefaultReadTimeout.Nanoseconds())
		if err := unix.SetsockoptTimeval(recvfd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &timeout); err != nil {
			log.Infof("set sockopt timeval failed: %s", err.Error())
			errch <- err
			return
		}

		for {
			buf := make([]byte, MaxUDPReceivedPacketSize)
			n, _, err := unix.Recvfrom(recvfd, buf, 0)
			if err != nil {
				log.Infof("unix recvfrom failed: %s", err.Error())
				errch <- err
				return
			}

			var ipHeader ipv4.Header
			if err := ipHeader.Parse(buf[:n]); err != nil || ipHeader.Protocol != 17 {
				continue
			}

			udpHeader := buf[ipHeader.Len:n]
			if int(binary.BigEndian.Uint16(udpHeader[0:2])) != dhcpv4.ServerPort ||
				int(binary.BigEndian.Uint16(udpHeader[2:4])) != dhcpv4.ClientPort {
				continue
			}

			payload := buf[ipHeader.Len+8 : ipHeader.Len+8+int(binary.BigEndian.Uint16(udpHeader[4:6]))]
			offer, err := dhcpv4.FromBytes(payload)
			if err != nil {
				log.Infof("dhcpv4 from bytes failed: %s", err.Error())
				errch <- err
				return
			}

			if offer.TransactionID != packet.TransactionID || offer.OpCode != dhcpv4.OpcodeBootReply {
				continue
			}

			if serverIP := offer.ServerIdentifier(); serverIP != nil {
				dhcpServers = append(dhcpServers, &DHCPServer{
					IPv4: serverIP.String(),
				})
			}
		}
	}(errCh)

	if err := unix.Sendto(sendfd, packetBytes, 0, &cli.remoteSocketAddr); err != nil {
		return nil, err
	}

	select {
	case err := <-errCh:
		if err == unix.EAGAIN {
			return nil, errors.New("timed out while listening for replies")
		}

		if err != nil {
			return nil, err
		}
	case <-time.After(DefaultReadTimeout):
		return dhcpServers, nil
	}

	return dhcpServers, nil
}

func makeBroadcastSocket(ifname string) (int, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_RAW, unix.IPPROTO_RAW)
	if err != nil {
		return fd, err
	}

	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		return fd, err
	}

	err = unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_HDRINCL, 1)
	if err != nil {
		return fd, err
	}

	err = unix.BindToDevice(fd, ifname)
	if err != nil {
		return fd, err
	}

	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_BROADCAST, 1)
	if err != nil {
		return fd, err
	}

	return fd, nil
}

func makeListeningSocketWithCustomPort(ifIndex int) (int, error) {
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM, int(htons(unix.ETH_P_IP)))
	if err != nil {
		return fd, err
	}

	llAddr := unix.SockaddrLinklayer{
		Ifindex:  ifIndex,
		Protocol: htons(unix.ETH_P_IP),
	}
	err = unix.Bind(fd, &llAddr)
	return fd, err
}

func htons(v uint16) uint16 {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], v)
	return binary.LittleEndian.Uint16(tmp[:])
}

func makeRawUDPPacket(payload []byte) ([]byte, error) {
	udp := make([]byte, 8)
	binary.BigEndian.PutUint16(udp[:2], uint16(dhcpv4.ClientPort))
	binary.BigEndian.PutUint16(udp[2:4], uint16(dhcpv4.ServerPort))
	binary.BigEndian.PutUint16(udp[4:6], uint16(8+len(payload)))
	binary.BigEndian.PutUint16(udp[6:8], 0)

	h := ipv4.Header{
		Version:  4,
		Len:      20,
		TotalLen: 20 + len(udp) + len(payload),
		TTL:      64,
		Protocol: 17,
		Dst:      net.IPv4bcast,
		Src:      net.IPv4zero,
	}
	ret, err := h.Marshal()
	if err != nil {
		return nil, err
	}
	ret = append(ret, udp...)
	ret = append(ret, payload...)
	return ret, nil
}
