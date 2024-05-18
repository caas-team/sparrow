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
	// UDP represents the UDP protocol
	UDP Protocol = iota
	// TCP represents the TCP protocol
	TCP
	// bufferSize represents the buffer size for the received data
	bufferSize = 1500
)

// Tracer represents a traceroute implementation
//
//go:generate moq -out traceroute_moq.go . Tracer
type Tracer interface {
	// Run performs a traceroute to the given address using the specified protocol and port
	Run(ctx context.Context, address string, port uint16) ([]Hop, error)
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

// Run performs a traceroute to the given address using the specified protocol and port
func (t *tracer) Run(ctx context.Context, address string, port uint16) ([]Hop, error) {
	if t.Protocol == UDP {
		return nil, errors.New("UDP protocol is not supported yet")
	}

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
			hop, err := t.doHop(ctx, destAddr, port, ttl)
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

// ErrClosingConn represents an error that occurred while closing a connection
type ErrClosingConn struct {
	Err error
}

func (e ErrClosingConn) Error() string {
	return fmt.Sprintf("error closing connection: %v", e.Err)
}

// Unwrap returns the wrapped error
func (e ErrClosingConn) Unwrap() error {
	return e.Err
}

// Is checks if the target error is an ErrClosingConn
func (e ErrClosingConn) Is(target error) bool {
	_, ok := target.(*ErrClosingConn)
	return ok
}

// doHop performs a single hop in the traceroute to the given address with the specified TTL.
func (t *tracer) doHop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (hop Hop, err error) {
	hop.Tracepoint = ttl
	network := "ip4:icmp"
	if destAddr.IP.To4() == nil {
		network = "ip6:ipv6-icmp"
	}

	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	icmpConn, err := icmp.ListenPacket(network, "0.0.0.0")
	if err != nil {
		return hop, fmt.Errorf("error creating ICMP listener: %w", err)
	}
	defer func() {
		cErr := icmpConn.Close()
		if cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	conn, err := net.DialIP(network, nil, destAddr)
	if err != nil {
		return hop, fmt.Errorf("error creating raw socket: %w", err)
	}
	defer func() {
		cErr := conn.Close()
		if cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	if network == "ip4:icmp" {
		pc := ipv4.NewPacketConn(conn)
		if err := pc.SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return hop, fmt.Errorf("error setting control message: %w", err)
		}
		if err := pc.SetTTL(ttl); err != nil {
			return hop, fmt.Errorf("error setting TTL: %w", err)
		}
	} else {
		pc := ipv6.NewPacketConn(conn)
		if err := pc.SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return hop, fmt.Errorf("error setting control message: %w", err)
		}
		if err := pc.SetHopLimit(ttl); err != nil {
			return hop, fmt.Errorf("error setting hop limit: %w", err)
		}
	}

	var icmpType icmp.Type
	if network == "ip4:icmp" {
		icmpType = ipv4.ICMPTypeEcho
	} else {
		icmpType = ipv6.ICMPTypeEchoRequest
	}

	wm := icmp.Message{
		Type: icmpType,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: ttl,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}

	wb, err := wm.Marshal(nil)
	if err != nil {
		return hop, fmt.Errorf("error marshalling ICMP message: %w", err)
	}

	start := time.Now()
	if _, err := conn.Write(wb); err != nil {
		return hop, fmt.Errorf("error sending packet: %w", err)
	}

	recvBuffer := make([]byte, bufferSize)
	icmpConn.SetReadDeadline(time.Now().Add(t.Timeout))

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
