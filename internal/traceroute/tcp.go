package traceroute

import (
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (t *tracer) hopTCP(destAddr *net.IPAddr, port uint16, ttl int) (hop Hop, err error) {
	conn, err := createTCPConn(destAddr, port, ttl)
	if err != nil {
		return hop, fmt.Errorf("error creating TCP connection: %w", err)
	}
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	if err = sendTCPPacket(conn); err != nil {
		return hop, fmt.Errorf("error sending TCP packet: %w", err)
	}

	icmpConn, err := icmp.ListenPacket(getICMPNetwork(destAddr), "")
	if err != nil {
		return hop, fmt.Errorf("error creating ICMP listener: %w", err)
	}
	defer func() {
		if cErr := icmpConn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	recvBuffer := make([]byte, bufferSize)
	icmpConn.SetReadDeadline(time.Now().Add(t.Timeout))

	hop, err = receiveICMPResponse(icmpConn, recvBuffer, start)
	hop.Tracepoint = ttl
	if err != nil {
		return hop, err
	}

	return hop, nil
}

// createTCPConn creates a TCP connection to the given address with the specified TTL
func createTCPConn(destAddr *net.IPAddr, port uint16, ttl int) (*net.TCPConn, error) {
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: destAddr.IP, Port: int(port)})
	if err != nil {
		return nil, err
	}

	if destAddr.IP.To4() != nil {
		pc := ipv4.NewConn(conn)
		if err := pc.SetTTL(ttl); err != nil {
			return nil, err
		}
	} else {
		pc := ipv6.NewConn(conn)
		if err := pc.SetHopLimit(ttl); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

// sendTCPPacket sends a TCP SYN packet to the given destination
func sendTCPPacket(conn *net.TCPConn) error {
	return conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
}

// getICMPNetwork returns the appropriate ICMP network based on the IP address version
func getICMPNetwork(destAddr *net.IPAddr) string {
	if destAddr.IP.To4() != nil {
		return "ip4:icmp"
	}
	return "ip6:ipv6-icmp"
}
