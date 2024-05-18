package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
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
			hop, err := t.performHop(ctx, destAddr, port, ttl)
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

// performHop performs a single hop in the traceroute to the given address with the specified TTL.
func (t *tracer) performHop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (Hop, error) {
	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	hop, err := t.doHop(ctx, destAddr, port, ttl)
	if err != nil {
		hop.IP = destAddr.IP
		hop.ReachedTarget = false
		hop.Error = err.Error()
		return hop, err
	}

	return hop, nil
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
	network := "tcp4"
	if destAddr.IP.To4() == nil {
		network = "tcp6"
	}

	dialer := &net.Dialer{
		Timeout: t.Timeout,
		ControlContext: func(_ context.Context, nw, _ string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				if nw == "tcp4" {
					err = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl)
				} else {
					err = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_UNICAST_HOPS, ttl)
				}
			})
		},
	}
	if err != nil {
		return hop, fmt.Errorf("error setting TTL: %w", err)
	}

	conn, err := dialer.DialContext(ctx, network, fmt.Sprintf("%s:%d", destAddr.String(), port))
	if err != nil {
		return hop, fmt.Errorf("error dialing: %w", err)
	}
	defer func() {
		cErr := conn.Close()
		if cErr != nil {
			err = errors.Join(err, &ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	_, err = conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	if err != nil {
		return hop, fmt.Errorf("error writing to connection: %w", err)
	}

	recvBuffer := make([]byte, bufferSize)
	err = conn.SetReadDeadline(time.Now().Add(t.Timeout))
	if err != nil {
		return hop, fmt.Errorf("error setting read deadline: %w", err)
	}

	n, err := conn.Read(recvBuffer)
	if err != nil {
		return hop, fmt.Errorf("error reading from connection: %w", err)
	}
	hop.Duration = time.Since(start)

	srcAddr := conn.RemoteAddr().(*net.TCPAddr)
	hop.IP = srcAddr.IP
	hop.ReachedTarget = n > 0

	return hop, nil
}
