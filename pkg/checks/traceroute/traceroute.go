// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package traceroute

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sys/unix"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// IPv4HeaderSize is the size of an IPv4 header in bytes
	IPv4HeaderSize = 20
	// IPv6HeaderSize is the size of an IPv6 header in bytes
	IPv6HeaderSize = 40
	// mtuSize is the maximum transmission unit size
	mtuSize = 1500
	// basePort is the starting port for the TCP connection
	basePort = 30000
	// portRange is the range of ports to generate a random port from
	portRange = 10000
)

// randomPort returns a random port in the interval [30_000, 40_000)
func randomPort() int {
	return rand.N(portRange) + basePort // #nosec G404 // math.rand is fine here, we're not doing encryption
}

// tcpHop attempts to connect to the target host using TCP with the specified TTL and timeout.
// It returns a [net.Conn], the port used for the connection, and an error if the connection failed.
func tcpHop(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (net.Conn, int, error) {
	span := trace.SpanFromContext(ctx)

	for {
		port := randomPort()

		// Dialer with control function to set IP_TTL
		dialer := net.Dialer{
			LocalAddr: &net.TCPAddr{
				Port: port,
			},
			Timeout: timeout,
			Control: func(_, _ string, c syscall.RawConn) error {
				var opErr error
				if err := c.Control(func(fd uintptr) {
					opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl) // #nosec G115 // The net package is safe to use
				}); err != nil {
					return err
				}
				return opErr
			},
		}

		span.AddEvent("Attempting TCP connection", trace.WithAttributes(
			attribute.String("remote_addr", addr.String()),
			attribute.Int("ttl", ttl),
			attribute.Int("port", port),
		))

		// Attempt to connect to the target host
		conn, err := dialer.DialContext(ctx, "tcp", addr.String())

		switch {
		case err == nil:
			span.AddEvent("TCP connection succeeded", trace.WithAttributes(
				attribute.Stringer("remote_addr", addr),
				attribute.Int("ttl", ttl),
				attribute.Int("port", port),
			))
			return conn, port, nil
		case errors.Is(err, unix.EADDRINUSE):
			// Address in use, retry by continuing the loop
			continue
		case errors.Is(err, unix.EHOSTUNREACH):
			// No route to host is a special error because of how tcp traceroute works
			// we are expecting the connection to fail because of TTL expiry
			span.SetStatus(codes.Error, "No route to host")
			span.AddEvent("No route to host", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
			logger.FromContext(ctx).DebugContext(ctx, "No route to host", "error", err.Error())
			return conn, port, err
		default:
			span.AddEvent("TCP connection failed", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return conn, port, err
		}
	}
}

// readIcmpMessage reads a packet from the provided [icmp.PacketConn]. If the packet is 'Time Exceeded',
// it reads the address of the router that dropped created the icmp packet. It also reads the source port
// from the payload and finds the source port used by the previous tcp connection. If any error is returned,
// an icmp packet was either not received, or the received packet was not a time exceeded.
func readIcmpMessage(ctx context.Context, icmpListener *icmp.PacketConn, timeout time.Duration) (int, net.Addr, error) {
	log := logger.FromContext(ctx)
	// Expected to fail due to TTL expiry, listen for ICMP response
	if err := icmpListener.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return 0, nil, fmt.Errorf("failed to set icmp read deadline: %w", err)
	}
	buffer := make([]byte, mtuSize)
	n, routerAddr, err := icmpListener.ReadFrom(buffer)
	if err != nil {
		// This is probably a timeout, so we can return an error
		return 0, nil, fmt.Errorf("failed to read from icmp connection: %w", err)
	}

	// Parse the ICMP message
	msg, err := icmp.ParseMessage(ipv4.ICMPTypeTimeExceeded.Protocol(), buffer[:n])
	if err != nil {
		return 0, nil, err
	}

	// Extract the TCP segment from the ICMP message
	var tcpSegment []byte
	switch msg.Type {
	case ipv4.ICMPTypeTimeExceeded:
		tcpSegment = msg.Body.(*icmp.TimeExceeded).Data[IPv4HeaderSize:]
	case ipv6.ICMPTypeTimeExceeded:
		tcpSegment = msg.Body.(*icmp.TimeExceeded).Data[IPv6HeaderSize:]
	default:
		log.DebugContext(ctx, "message is not 'Time Exceeded'", "type", msg.Type.Protocol())
		return 0, nil, errors.New("message is not 'Time Exceeded'")
	}

	// Extract the source port from the TCP segment
	destPort := int(tcpSegment[0])<<8 + int(tcpSegment[1])

	return destPort, routerAddr, nil
}

// TraceRoute performs a traceroute to the specified host using TCP and listens for ICMP Time Exceeded messages using ICMP.
func TraceRoute(ctx context.Context, cfg tracerouteConfig) (map[int][]Hop, error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("tracer.traceroute")
	ctx, sp := tracer.Start(ctx, "TraceRoute", trace.WithAttributes(
		attribute.String("target", cfg.Dest),
		attribute.Int("port", cfg.Port),
		attribute.Int("max_hops", cfg.MaxHops),
		attribute.Stringer("timeout", cfg.Timeout),
	))
	defer sp.End()

	// Maps ttl -> attempted hops for that ttl
	hops := make(map[int][]Hop)
	log := logger.FromContext(ctx).With("target", cfg.Dest)

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", cfg.Dest, cfg.Port))
	if err != nil {
		sp.SetStatus(codes.Error, err.Error())
		sp.RecordError(err)
		log.ErrorContext(ctx, "failed to resolve target name", "err", err)
		return nil, err
	}

	// if we don't add the +1, this causes issues, when the user does not want to retry,
	// since the channel's size would be zero, blocking all threads from sending
	queueSize := cfg.MaxHops * (1 + cfg.Rc.Count)
	results := make(chan Hop, queueSize)
	var wg sync.WaitGroup

	for ttl := 1; ttl <= cfg.MaxHops; ttl++ {
		wg.Add(1)
		go func(ttl int) {
			c, hopSpan := tracer.Start(ctx, addr.String(), trace.WithAttributes(
				attribute.Int("ttl", ttl),
			))
			defer wg.Done()
			defer hopSpan.End()

			l := log.With("ttl", ttl)
			logctx := logger.IntoContext(c, l)

			retry := 0
			retryErr := helper.Retry(func(ctx context.Context) error {
				defer func() {
					retry++
				}()
				hopSpan.AddEvent("Attempting to hop", trace.WithAttributes(
					attribute.Int("ttl", ttl),
					attribute.Int("retry", retry),
				))

				hop, hErr := doHop(ctx, addr, ttl, cfg.Timeout)
				if hop != nil {
					results <- *hop
				}
				if hErr != nil {
					l.ErrorContext(ctx, "Failed to hop", "err", hErr)
					hopSpan.SetStatus(codes.Error, hErr.Error())
					hopSpan.RecordError(hErr)
					return hErr
				}

				if !hop.Reached {
					hopSpan.SetName(hop.Addr.String())
					l.DebugContext(ctx, "Failed to reach target, retrying")
					return errors.New("failed to reach target")
				}
				return nil
			}, cfg.Rc)(logctx)
			if retryErr != nil {
				l.DebugContext(ctx, "Traceroute could not reach target")
				if !errors.Is(err, syscall.EHOSTUNREACH) {
					hopSpan.SetStatus(codes.Error, retryErr.Error())
					hopSpan.RecordError(err)
				}
				return
			}
			hopSpan.SetStatus(codes.Ok, "Hop succeeded")
		}(ttl)
	}

	wg.Wait()
	close(results)

	// Collect and log hops
	for r := range results {
		hops[r.Ttl] = append(hops[r.Ttl], r)
	}
	logHops(ctx, hops)

	sp.AddEvent("TraceRoute completed", trace.WithAttributes(
		attribute.Int("hops_count", len(hops)),
	))
	return hops, nil
}

// doHop performs a hop to the given address with the specified TTL and timeout.
// It returns a Hop struct containing the latency, TTL, address, and other details of the hop.
func doHop(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (*Hop, error) {
	span := trace.SpanFromContext(ctx)
	log := logger.FromContext(ctx)
	canIcmp, icmpListener, err := newIcmpListener()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		log.ErrorContext(ctx, "Failed to open ICMP socket", "err", err.Error())
		return nil, err
	}
	defer closeIcmpListener(canIcmp, icmpListener)

	start := time.Now()
	conn, clientPort, err := tcpHop(ctx, addr, ttl, timeout)
	latency := time.Since(start)

	span.SetAttributes(attribute.Int("ttl", ttl), attribute.Stringer("addr", addr))
	if err == nil {
		hop := handleTcpSuccess(conn, addr, ttl, latency)
		span.AddEvent("Hop succeeded", trace.WithAttributes(
			attribute.String("hop_name", hop.Name),
			attribute.Stringer("hop_addr", hop.Addr),
			attribute.Stringer("latency", latency),
		))
		return hop, nil
	}

	if !canIcmp {
		span.AddEvent("ICMP socket not available")
		log.DebugContext(ctx, "No permission for icmp socket")
		return &Hop{
			Latency: latency,
			Ttl:     ttl,
			Reached: false,
		}, nil
	}

	hop := handleIcmpResponse(ctx, icmpListener, clientPort, ttl, timeout)
	hop.Latency = latency
	if !hop.Reached {
		span.AddEvent("ICMP hop not reached", trace.WithAttributes(
			attribute.String("hop_name", hop.Name),
			attribute.Stringer("hop_addr", hop.Addr),
			attribute.Stringer("latency", latency),
		))
		return &hop, nil
	}

	span.AddEvent("ICMP hop reached", trace.WithAttributes(
		attribute.String("hop_name", hop.Name),
		attribute.Stringer("hop_addr", hop.Addr),
		attribute.Stringer("latency", latency),
	))
	return &hop, nil
}

// newIcmpListener creates a new ICMP listener and returns a boolean indicating if the necessary permissions were granted.
func newIcmpListener() (bool, *icmp.PacketConn, error) {
	icmpListener, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		if !errors.Is(err, unix.EPERM) {
			return false, nil, err
		}
		return false, nil, nil
	}
	return true, icmpListener, nil
}

// closeIcmpListener closes the ICMP listener if it is not nil and the permissions were granted.
func closeIcmpListener(canIcmp bool, icmpListener *icmp.PacketConn) {
	if canIcmp && icmpListener != nil {
		icmpListener.Close() // #nosec G104
	}
}

// newHopAddress creates a new HopAddress from a [net.Addr].
func newHopAddress(addr net.Addr) HopAddress {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return HopAddress{
			IP:   addr.IP.String(),
			Port: addr.Port,
		}
	case *net.TCPAddr:
		return HopAddress{
			IP:   addr.IP.String(),
			Port: addr.Port,
		}
	case *net.IPAddr:
		return HopAddress{
			IP: addr.IP.String(),
		}
	default:
		return HopAddress{}
	}
}

// handleTcpSuccess handles a successful TCP connection by closing the connection and returning a Hop struct.
func handleTcpSuccess(conn net.Conn, addr net.Addr, ttl int, latency time.Duration) *Hop {
	conn.Close() // #nosec G104

	ipaddr := ipFromAddr(addr)
	names, _ := net.LookupAddr(ipaddr.String()) // we don't care about this lookup failing

	name := ""
	if len(names) >= 1 {
		name = names[0]
	}

	return &Hop{
		Latency: latency,
		Ttl:     ttl,
		Addr:    newHopAddress(addr),
		Name:    name,
		Reached: true,
	}
}

// handleIcmpResponse attempts to read a time exceeded packet that matches clientPort until timeout is reached
// if an error occurs while reading from the socket, handleIcmpResponse will silently fail and return a hop with hop.Reached=false
func handleIcmpResponse(ctx context.Context, icmpListener *icmp.PacketConn, clientPort, ttl int, timeout time.Duration) Hop {
	log := logger.FromContext(ctx)
	deadline := time.Now().Add(timeout)

	for time.Now().Unix() < deadline.Unix() {
		log.DebugContext(ctx, "Reading ICMP message")
		gotPort, addr, err := readIcmpMessage(ctx, icmpListener, timeout)
		if err != nil {
			log.DebugContext(ctx, "Failed to read ICMP message", "err", err.Error())
			continue
		}

		// Check if the destination port matches our dialer's source port
		if gotPort == clientPort {
			ipaddr := ipFromAddr(addr)
			names, _ := net.LookupAddr(ipaddr.String()) // we don't really care if this lookup works, so ignore the error

			name := ""
			if len(names) >= 1 {
				name = names[0]
			}

			return Hop{
				Ttl:  ttl,
				Addr: newHopAddress(addr),
				Name: name,
			}
		}
	}

	log.DebugContext(ctx, "Deadline reached")
	return Hop{
		Ttl: ttl,
	}
}

// ipFromAddr returns the IP address from a [net.Addr].
func ipFromAddr(remoteAddr net.Addr) net.IP {
	switch addr := remoteAddr.(type) {
	case *net.UDPAddr:
		return addr.IP
	case *net.TCPAddr:
		return addr.IP
	case *net.IPAddr:
		return addr.IP
	}
	return nil
}

// Hop represents a single hop in a traceroute
type Hop struct {
	Latency time.Duration `json:"latency" yaml:"latency" mapstructure:"latency"`
	Addr    HopAddress    `json:"addr" yaml:"addr" mapstructure:"addr"`
	Name    string        `json:"name" yaml:"name" mapstructure:"name"`
	Ttl     int           `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
	Reached bool          `json:"reached" yaml:"reached" mapstructure:"reached"`
}

// HopAddress represents an IP address and port
type HopAddress struct {
	IP   string `json:"ip" yaml:"ip" mapstructure:"ip"`
	Port int    `json:"port" yaml:"port" mapstructure:"port"`
}

// String returns the string representation of the [HopAddress].
func (a HopAddress) String() string {
	if a.Port != 0 {
		return fmt.Sprintf("%s:%d", a.IP, a.Port)
	}
	return a.IP
}

// logHops logs the hops in the mapHops map
func logHops(ctx context.Context, mapHops map[int][]Hop) {
	log := logger.FromContext(ctx)

	keys := []int{}
	for k := range mapHops {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, key := range keys {
		for _, hop := range mapHops[key] {
			out := fmt.Sprintf("%d %s %s %v ", key, hop.Addr.String(), hop.Name, hop.Latency)
			if hop.Reached {
				out += "( Reached )"
			}
			log.DebugContext(ctx, out)
		}
	}
}
