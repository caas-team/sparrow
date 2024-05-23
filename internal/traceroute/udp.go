package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var _ hopper = (*udpHopper)(nil)

type udpHopper struct{ *icmpHopper }

func (h *udpHopper) Hop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (hop Hop, err error) {
	log := logger.FromContext(ctx)
	network, _ := h.resolveType(destAddr)
	recvConn, err := icmp.ListenPacket(network, "")
	if err != nil {
		log.ErrorContext(ctx, "Error creating ICMP listener", "error", err)
		return hop, fmt.Errorf("error creating ICMP listener: %w", err)
	}
	log.DebugContext(ctx, "ICMP listener created", "address", destAddr.String(), "port", port, "ttl", ttl)
	defer func() {
		if cErr := recvConn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	conn, err := h.newConn(destAddr, port, ttl)
	if err != nil {
		log.ErrorContext(ctx, "Error creating UDP connection", "error", err)
		return hop, fmt.Errorf("error creating UDP connection: %w", err)
	}
	log.DebugContext(ctx, "UDP connection created", "address", destAddr.String(), "port", port, "ttl", ttl)
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	recvBuffer := make([]byte, bufferSize)
	err = recvConn.SetReadDeadline(time.Now().Add(h.Timeout))
	if err != nil {
		log.ErrorContext(ctx, "Error setting read deadline", "error", err)
		return hop, fmt.Errorf("error setting read deadline: %w", err)
	}

	hop, err = h.receive(recvConn, recvBuffer, start)
	hop.Tracepoint = ttl
	if err != nil {
		log.ErrorContext(ctx, "Error receiving ICMP message", "error", err)
		return hop, err
	}
	log.DebugContext(ctx, "ICMP message received", "address", destAddr.String(), "port", port, "ttl", ttl, "hop", hop)

	return hop, nil
}

// newConn creates a UDP connection to the given address with the specified TTL
func (*udpHopper) newConn(destAddr *net.IPAddr, port uint16, ttl int) (*net.UDPConn, error) {
	// Unfortunately, the net package does not provide a context-aware DialUDP function
	// TODO: Switch to the net.DialUDPContext function as soon as https://github.com/golang/go/issues/49097 is implemented
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: destAddr.IP, Port: int(port)})
	if err != nil {
		return nil, err
	}

	if destAddr.IP.To4() != nil {
		p := ipv4.NewConn(conn)
		if err := p.SetTTL(ttl); err != nil {
			return nil, err
		}
	} else {
		p := ipv6.NewConn(conn)
		if err := p.SetHopLimit(ttl); err != nil {
			return nil, err
		}
	}

	return conn, nil
}
