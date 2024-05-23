package traceroute

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"syscall"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"golang.org/x/sys/unix"
)

var _ hopper = (*tcpHopper)(nil)

type tcpHopper struct{ *tracer }

func (h *tcpHopper) Hop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (hop Hop, err error) {
	log := logger.FromContext(ctx)

	start := time.Now()
	conn, err := h.newConn(ctx, destAddr, port, ttl)
	if err != nil {
		log.InfoContext(ctx, "Connection could not be established, packet likely dropped by a router", "address", destAddr.String(), "port", port, "ttl", ttl, "error", err)
		hop.Tracepoint = ttl
		return hop, nil
	}
	log.DebugContext(ctx, "TCP connection created", "address", destAddr.String(), "port", port, "ttl", ttl)
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			log.ErrorContext(ctx, "Error closing TCP connection", "error", cErr)
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	hop, err = h.receive(conn, h.Timeout, start)
	hop.Tracepoint = ttl
	log.DebugContext(ctx, "TCP response received", "address", destAddr.String(), "port", port, "ttl", ttl, "hop", hop, "error", err)
	return hop, err
}

// newConn creates a TCP connection to the given address with the specified TTL
func (*tcpHopper) newConn(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (*net.TCPConn, error) {
	d := net.Dialer{
		ControlContext: func(ctx context.Context, _, _ string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				log := logger.FromContext(ctx)
				if destAddr.IP.To4() != nil {
					if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl); err != nil {
						log.ErrorContext(ctx, "Error setting IP_TTL", "error", err)
					}
					return
				}
				if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_UNICAST_HOPS, ttl); err != nil {
					log.ErrorContext(ctx, "Error setting IPV6_UNICAST_HOPS", "error", err)
				}
			})
		},
	}

	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(destAddr.String(), strconv.Itoa(int(port))))
	if err != nil {
		return nil, fmt.Errorf("error dialing TCP connection: %w", err)
	}

	return conn.(*net.TCPConn), nil
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
