package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
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

// Run performs a traceroute to the given address using the specified protocol
func (t *tracer) Run(ctx context.Context, address string, port uint16) ([]Hop, error) {
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
			hop, err := t.hop(ctx, destAddr, port, ttl)
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

// hop performs a single hop in the traceroute to the given address with the specified TTL.
func (t *tracer) hop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (Hop, error) {
	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return Hop{}, ctx.Err()
	default:
		switch t.Protocol {
		case ICMP:
			return t.hopICMP(destAddr, ttl)
		case UDP:
			return Hop{}, errors.New("UDP not supported yet")
		case TCP:
			return t.hopTCP(destAddr, port, ttl)
		default:
			return Hop{}, errors.New("protocol not supported")
		}
	}
}
