package traceroute

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (t *tracer) hopICMP(destAddr *net.IPAddr, ttl int) (hop Hop, err error) {
	network, icmpType := getNetworkAndICMPType(destAddr)
	icmpConn, err := icmp.ListenPacket(network, "")
	if err != nil {
		return hop, fmt.Errorf("error creating ICMP listener: %w", err)
	}
	defer func() {
		if cErr := icmpConn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	conn, err := createRawConn(network, destAddr, ttl)
	if err != nil {
		return hop, fmt.Errorf("error creating raw socket: %w", err)
	}
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	if err = sendICMPMessage(conn, icmp.Message{
		Type: icmpType,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: ttl,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}); err != nil {
		return hop, fmt.Errorf("error sending ICMP message: %w", err)
	}

	recvBuffer := make([]byte, bufferSize)
	err = icmpConn.SetReadDeadline(time.Now().Add(t.Timeout))
	if err != nil {
		return hop, fmt.Errorf("error setting read deadline: %w", err)
	}

	hop, err = receiveICMPResponse(icmpConn, recvBuffer, start)
	hop.Tracepoint = ttl
	if err != nil {
		return hop, err
	}

	return hop, nil
}

// getNetworkAndICMPType returns the network and ICMP type based on the destination address
func getNetworkAndICMPType(destAddr *net.IPAddr) (string, icmp.Type) {
	if destAddr.IP.To4() != nil {
		return "ip4:icmp", ipv4.ICMPTypeEcho
	}
	return "ip6:ipv6-icmp", ipv6.ICMPTypeEchoRequest
}

// createRawConn creates a raw connection to the given address with the specified TTL
func createRawConn(network string, destAddr *net.IPAddr, ttl int) (*net.IPConn, error) {
	conn, err := net.DialIP(network, nil, destAddr)
	if err != nil {
		return nil, err
	}

	if network == "ip4:icmp" {
		pc := ipv4.NewPacketConn(conn)
		if err := pc.SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return nil, err
		}
		if err := pc.SetTTL(ttl); err != nil {
			return nil, err
		}
	} else {
		pc := ipv6.NewPacketConn(conn)
		if err := pc.SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return nil, err
		}
		if err := pc.SetHopLimit(ttl); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

// sendICMPMessage sends an ICMP message to the given connection
func sendICMPMessage(conn *net.IPConn, wm icmp.Message) error {
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}

	_, err = conn.Write(wb)
	return err
}

// receiveICMPResponse reads the response from the ICMP connection
func receiveICMPResponse(icmpConn *icmp.PacketConn, recvBuffer []byte, start time.Time) (Hop, error) {
	hop := Hop{}
	n, peer, err := icmpConn.ReadFrom(recvBuffer)
	if err != nil {
		return hop, fmt.Errorf("error reading from ICMP connection: %w", err)
	}
	hop.Duration = time.Since(start).Seconds()

	rm, err := icmp.ParseMessage(1, recvBuffer[:n])
	if err != nil {
		return hop, fmt.Errorf("error parsing ICMP message: %w", err)
	}

	switch rm.Type {
	case ipv4.ICMPTypeTimeExceeded, ipv6.ICMPTypeTimeExceeded:
		hop.IP = peer.(*net.IPAddr).IP
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		hop.IP = peer.(*net.IPAddr).IP
		hop.ReachedTarget = true
	default:
		return hop, fmt.Errorf("unexpected ICMP message type: %v", rm.Type)
	}

	return hop, nil
}
