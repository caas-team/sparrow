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
	"golang.org/x/sys/unix"
)

var _ hopper = (*icmpHopper)(nil)

type icmpHopper struct{ *tracer }

func (h *icmpHopper) Hop(ctx context.Context, destAddr *net.IPAddr, _ uint16, ttl int) (hop Hop, err error) {
	log := logger.FromContext(ctx)
	network, typ := h.resolveType(destAddr)
	log.DebugContext(ctx, "Resolved network and ICMP type", "network", network, "type", typ)

	recvConn, err := icmp.ListenPacket(network, "")
	if err != nil {
		log.ErrorContext(ctx, "Error creating ICMP listener", "error", err)
		return hop, fmt.Errorf("error creating ICMP listener: %w", err)
	}
	log.DebugContext(ctx, "ICMP listener created", "address", recvConn.LocalAddr().String())
	defer func() {
		if cErr := recvConn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	conn, err := h.newConn(network, destAddr, ttl)
	if err != nil {
		log.ErrorContext(ctx, "Error creating raw socket", "error", err)
		return hop, fmt.Errorf("error creating raw socket: %w", err)
	}
	log.DebugContext(ctx, "Raw socket created", "address", destAddr.String(), "ttl", ttl)
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	if err = h.send(conn, icmp.Message{
		Type: typ,
		Code: 0,
		Body: &icmp.Echo{
			ID: unix.Getpid() & 0xffff, Seq: ttl,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}); err != nil {
		log.ErrorContext(ctx, "Error sending ICMP message", "error", err)
		return hop, fmt.Errorf("error sending ICMP message: %w", err)
	}
	log.DebugContext(ctx, "ICMP message sent", "address", destAddr.String(), "ttl", ttl)

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
	log.DebugContext(ctx, "ICMP message received", "address", destAddr.String(), "ttl", ttl, "hop", hop)

	return hop, nil
}

// resolveType returns the network and ICMP type based on the destination address
func (*icmpHopper) resolveType(destAddr *net.IPAddr) (network string, typ icmp.Type) {
	if destAddr.IP.To4() != nil {
		return "ip4:icmp", ipv4.ICMPTypeEcho
	}
	return "ip6:ipv6-icmp", ipv6.ICMPTypeEchoRequest
}

// newConn creates a raw connection to the given address with the specified TTL
func (*icmpHopper) newConn(network string, destAddr *net.IPAddr, ttl int) (*net.IPConn, error) {
	// Unfortunately, the net package does not provide a context-aware DialIP function
	// TODO: Switch to the net.DialIPContext function as soon as https://github.com/golang/go/issues/49097 is implemented
	conn, err := net.DialIP(network, nil, destAddr)
	if err != nil {
		return nil, err
	}

	if network == "ip4:icmp" {
		p := ipv4.NewPacketConn(conn)
		if err := p.SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return nil, err
		}
		if err := p.SetTTL(ttl); err != nil {
			return nil, err
		}
	} else {
		p := ipv6.NewPacketConn(conn)
		if err := p.SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return nil, err
		}
		if err := p.SetHopLimit(ttl); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

// send sends an ICMP message to the given connection
func (*icmpHopper) send(conn *net.IPConn, msg icmp.Message) error {
	b, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	_, err = conn.Write(b)
	return err
}

// receive reads the response from the ICMP connection
func (*icmpHopper) receive(conn *icmp.PacketConn, buffer []byte, start time.Time) (Hop, error) {
	hop := Hop{}
	n, peer, err := conn.ReadFrom(buffer)
	if err != nil {
		return hop, fmt.Errorf("error reading from ICMP connection: %w", err)
	}
	hop.Duration = time.Since(start).Seconds()

	pm, err := icmp.ParseMessage(1, buffer[:n])
	if err != nil {
		return hop, fmt.Errorf("error parsing ICMP message: %w", err)
	}

	switch pm.Type {
	case ipv4.ICMPTypeTimeExceeded, ipv6.ICMPTypeTimeExceeded:
		hop.IP = peer.(*net.IPAddr).IP
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		hop.IP = peer.(*net.IPAddr).IP
		hop.ReachedTarget = true
	default:
		return hop, fmt.Errorf("unexpected ICMP message type: %v", pm.Type)
	}

	return hop, nil
}
