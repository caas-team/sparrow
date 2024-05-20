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

func (t *tracer) hopTCP(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (hop Hop, err error) {
	log := logger.FromContext(ctx)
	conn, err := createTCPConn(ctx, destAddr, port, ttl)
	if err != nil {
		log.Error("Error creating TCP connection", "error", err)
		return hop, fmt.Errorf("error creating TCP connection: %w", err)
	}
	log.Debug("TCP connection created", "address", destAddr.String(), "port", port, "ttl", ttl)
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			log.Error("Error closing TCP connection", "error", cErr)
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	if err = sendTCPPacket(ctx, conn); err != nil {
		log.Error("Error sending TCP packet", "error", err)
		return hop, fmt.Errorf("error sending TCP packet: %w", err)
	}
	log.Debug("TCP packet sent", "address", destAddr.String(), "port", port, "ttl", ttl)

	hop, err = receiveTCPResponse(ctx, conn, t.Timeout, start)
	hop.Tracepoint = ttl
	log.Debug("TCP response received", "address", destAddr.String(), "port", port, "ttl", ttl, "hop", hop, "error", err)
	return hop, err
}

// createTCPConn creates a TCP connection to the given address with the specified TTL
func createTCPConn(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (*net.TCPConn, error) {
	log := logger.FromContext(ctx)
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: destAddr.IP, Port: int(port)})
	if err != nil {
		log.Error("Error dialing TCP connection", "error", err)
		return nil, err
	}

	if destAddr.IP.To4() != nil {
		pc := ipv4.NewConn(conn)
		if err := pc.SetTTL(ttl); err != nil {
			log.Error("Error setting TTL on IPv4 connection", "error", err)
			return nil, err
		}
	} else {
		pc := ipv6.NewConn(conn)
		if err := pc.SetHopLimit(ttl); err != nil {
			log.Error("Error setting hop limit on IPv6 connection", "error", err)
			return nil, err
		}
	}
	return conn, nil
}

// sendTCPPacket sends a TCP SYN packet to the given destination
func sendTCPPacket(ctx context.Context, conn *net.TCPConn) error {
	log := logger.FromContext(ctx)
	err := conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		log.Error("Error setting write deadline", "error", err)
		return fmt.Errorf("error setting write deadline: %w", err)
	}

	_, err = conn.Write([]byte("HELLO-R-U-THERE"))
	if err != nil {
		log.Error("Error writing TCP packet", "error", err)
		return fmt.Errorf("error writing TCP packet: %w", err)
	}

	return nil
}

// receiveTCPResponse waits for a TCP response to the sent SYN packet
func receiveTCPResponse(ctx context.Context, conn *net.TCPConn, timeout time.Duration, start time.Time) (Hop, error) {
	hop := Hop{}
	log := logger.FromContext(ctx)
	err := conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		log.Error("Error setting read deadline", "error", err)
		return hop, fmt.Errorf("error setting read deadline: %w", err)
	}

	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err != nil {
		// Timeout means the TTL expired
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			hop.Duration = time.Since(start).Seconds()
			hop.IP = conn.RemoteAddr().(*net.TCPAddr).IP
			return hop, nil
		}
		// EOF means the target sent a TCP RST response
		if err == io.EOF {
			hop.Duration = time.Since(start).Seconds()
			hop.IP = conn.RemoteAddr().(*net.TCPAddr).IP
			hop.ReachedTarget = true
			return hop, nil
		}
		log.Error("Error reading TCP response", "error", err)
		return hop, fmt.Errorf("error reading TCP response: %w", err)
	}

	hop.Duration = time.Since(start).Seconds()
	hop.IP = conn.RemoteAddr().(*net.TCPAddr).IP
	hop.ReachedTarget = true
	return hop, nil
}
