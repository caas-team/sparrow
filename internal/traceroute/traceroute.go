package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var _ Tracer = (*tracer)(nil)

// Protocol defines the protocol used for the traceroute
type Protocol int

const (
	// ICMP represents the ICMP protocol
	ICMP Protocol = iota
	// UDP represents the UDP protocol
	UDP
	// TCP represents the TCP protocol
	TCP
	// bufferSize represents the buffer size for the received data
	bufferSize = 1500
)

// Tracer represents a traceroute implementation
//
//go:generate moq -out traceroute_moq.go . Tracer
type Tracer interface {
	// Run performs a traceroute to the given address using the specified protocol
	Run(ctx context.Context, address string) ([]Hop, error)
}

// tracer implements the Tracer interface
// It is used to perform a traceroute to a given address
type tracer struct {
	// MaxHops defines the maximum number of hops to perform the traceroute
	MaxHops int
	// Timeout defines the maximum time to wait for a response from each hop
	Timeout time.Duration
	// Protocol defines the Protocol to use for the traceroute
	Protocol Protocol
}

// New creates a new tracer instance with the given configurations
func New(maxHops int, timeout time.Duration, protocol Protocol) Tracer {
	if maxHops <= 0 {
		maxHops = 30
	}

	return &tracer{
		MaxHops:  maxHops,
		Timeout:  timeout,
		Protocol: protocol,
	}
}

// Hop represents the result of a single hop in the traceroute
type Hop struct {
	// Tracepoint represents the hop number
	Tracepoint int
	// IP represents the IP address of the hop
	IP net.IP
	// Error represents the error that occurred during the hop
	Error string
	// Duration represents the time it took to reach the hop
	Duration time.Duration
	// ReachedTarget indicates whether the target was reached with this hop
	ReachedTarget bool
}

// Run performs a traceroute to the given address using the specified protocol
func (t *tracer) Run(ctx context.Context, address string) ([]Hop, error) {
	destAddr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return nil, fmt.Errorf("error resolving IP address: %w", err)
	}

	var hops []Hop
	for ttl := 1; ttl <= t.MaxHops; ttl++ {
		select {
		case <-ctx.Done():
			return hops, ctx.Err()
		default:
			hop, err := t.doHop(ctx, destAddr, ttl)
			if err != nil {
				hop.IP = destAddr.IP
				hop.ReachedTarget = false
				hop.Error = err.Error()
				return append(hops, hop), err
			}
			hops = append(hops, hop)
			if hop.ReachedTarget {
				return hops, nil
			}
		}
	}

	return hops, nil
}

// doHop performs a single hop in the traceroute to the given address with the specified TTL.
func (t *tracer) doHop(_ context.Context, destAddr *net.IPAddr, ttl int) (hop Hop, err error) {
	// TODO: use the context to timeout the traceroute if it takes too long
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
	icmpConn.SetReadDeadline(time.Now().Add(t.Timeout))

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
	hop.Duration = time.Since(start)

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
