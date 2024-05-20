package traceroute

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var _ hopper = (*tcpHopper)(nil)

type tcpHopper struct{ *tracer }

func (h *tcpHopper) Hop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (hop Hop, err error) {
	log := logger.FromContext(ctx)
	conn, err := h.newConn(destAddr, port, ttl)
	if err != nil {
		log.ErrorContext(ctx, "Error creating TCP connection", "error", err)
		return hop, fmt.Errorf("error creating TCP connection: %w", err)
	}
	log.DebugContext(ctx, "TCP connection created", "address", destAddr.String(), "port", port, "ttl", ttl)
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			log.ErrorContext(ctx, "Error closing TCP connection", "error", cErr)
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	if err = h.sendSYN(conn); err != nil {
		log.ErrorContext(ctx, "Error sending TCP packet", "error", err)
		return hop, fmt.Errorf("error sending TCP packet: %w", err)
	}
	log.DebugContext(ctx, "TCP packet sent", "address", destAddr.String(), "port", port, "ttl", ttl)

	hop, err = h.receive(conn, h.Timeout, start)
	hop.Tracepoint = ttl
	log.DebugContext(ctx, "TCP response received", "address", destAddr.String(), "port", port, "ttl", ttl, "hop", hop, "error", err)
	return hop, err
}

// newConn creates a TCP connection to the given address with the specified TTL
func (*tcpHopper) newConn(destAddr *net.IPAddr, port uint16, ttl int) (*net.TCPConn, error) {
	// Unfortunately, the net package does not provide a context-aware DialTCP function
	// TODO: Switch to the net.DialTCPContext function as soon as https://github.com/golang/go/issues/49097 is implemented
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: destAddr.IP, Port: int(port)})
	if err != nil {
		return nil, fmt.Errorf("error dialing TCP connection: %w", err)
	}

	if destAddr.IP.To4() != nil {
		p := ipv4.NewConn(conn)
		if err := p.SetTTL(ttl); err != nil {
			return nil, fmt.Errorf("error setting TTL on IPv4 connection: %w", err)
		}
	} else {
		p := ipv6.NewConn(conn)
		if err := p.SetHopLimit(ttl); err != nil {
			return nil, fmt.Errorf("error setting hop limit on IPv6 connection: %w", err)
		}
	}
	return conn, nil
}

// sendSYN writes a TCP SYN packet to the given connection's file descriptor
func (*tcpHopper) sendSYN(conn *net.TCPConn) error {
	err := conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return fmt.Errorf("error setting write deadline: %w", err)
	}

	// To initiate a TCP connection, we need to send a SYN packet
	// In this case we want this to be as small as possible, so we send a single byte
	_, err = conn.Write([]byte{0})
	if err != nil {
		return fmt.Errorf("error writing TCP packet: %w", err)
	}

	return nil
}

// receive waits for a TCP response to the sent SYN packet
func (*tcpHopper) receive(conn *net.TCPConn, timeout time.Duration, start time.Time) (Hop, error) {
	err := conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return Hop{}, fmt.Errorf("error setting read deadline: %w", err)
	}

	_, err = io.ReadAll(conn)
	if err != nil {
		// Timeout means the TTL expired and the packet was dropped by a router
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			return Hop{Duration: time.Since(start).Seconds()}, nil
		}
		return Hop{}, fmt.Errorf("error reading TCP response: %w", err)
	}

	return Hop{
		Duration:      time.Since(start).Seconds(),
		IP:            conn.RemoteAddr().(*net.TCPAddr).IP,
		ReachedTarget: true,
	}, nil
}
