package traceroute

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"slices"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	IPv4HeaderSize = 20
	mtuSize        = 1500 // Standard MTU size
)

// randomPort returns a random port in the interval [ 30_000, 40_000 [
//
//nolint:all
func randomPort() int {
	return rand.Intn(10_000) + 30_000 // #nosec G404 // math.rand is fine here, we're not doing encryption
}

func tcpHop(addr net.Addr, ttl int, timeout time.Duration) (net.Conn, int, error) {
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
					opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl)
				}); err != nil {
					return err
				}
				return opErr
			},
		}

		// Attempt to connect to the target host
		conn, err := dialer.Dial("tcp", addr.String())
		if !errors.Is(err, unix.EADDRINUSE) {
			return conn, port, err
		}
	}
}

// readIcmpMessage reads a packet from the provided icmp Connection. If the packet is 'Time Exceeded',
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
		// we probably timed out so return
		return 0, nil, fmt.Errorf("failed to read from icmp connection: %w", err)
	}

	// Parse the ICMP message
	msg, err := icmp.ParseMessage(ipv4.ICMPTypeTimeExceeded.Protocol(), buffer[:n])
	if err != nil {
		return 0, nil, err
	}

	// Ensure the message is an ICMP Time Exceeded message
	if msg.Type != ipv4.ICMPTypeTimeExceeded && msg.Type != ipv6.ICMPTypeTimeExceeded {
		log.Debug("message is not 'Time Exceeded'", "type", msg.Type.Protocol())
		return 0, nil, errors.New("message is not 'Time Exceeded'")
	}

	// The first 20 bytes of Data are the IP header, so the TCP segment starts at byte 20
	tcpSegment := msg.Body.(*icmp.TimeExceeded).Data[IPv4HeaderSize:]

	// Extract the source port from the TCP segment
	destPort := int(tcpSegment[0])<<8 + int(tcpSegment[1])

	return destPort, routerAddr, nil
}

// TraceRoute performs a traceroute to the specified host using TCP and listens for ICMP Time Exceeded messages using ICMP.
func TraceRoute(ctx context.Context, cfg tracerouteConfig) (map[int][]Hop, error) {
	// maps ttl -> attempted hops for that ttl
	hops := make(map[int][]Hop)
	log := logger.FromContext(ctx).With("target", cfg.Dest)

	timeoutDuration := time.Duration(cfg.Timeout) * time.Second

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", cfg.Dest, cfg.Port))
	if err != nil {
		log.Error("failed to resolve target name", "err", err.Error())
		return nil, err
	}

	// if we don't add the +1, this causes issues, when the user does not want to retry,
	// since the channels size would be zero, blocking all threads from sending
	queueSize := cfg.MaxHops * (1 + cfg.Rc.Count)
	results := make(chan Hop, queueSize)
	var wg sync.WaitGroup

	for ttl := 1; ttl <= cfg.MaxHops; ttl++ {
		wg.Add(1)
		go func(ttl int) {
			defer wg.Done()
			err := helper.Retry(func(_ context.Context) error {
				hop, err := traceroute(ctx, addr, ttl, timeoutDuration)
				if hop != nil {
					results <- *hop
				}
				if err != nil {
					log.Error("traceroute failed", "err", err.Error(), "ttl", ttl)
					return err
				}
				if !hop.Reached {
					log.Debug("failed to reach target, retrying", "ttl", ttl)
					return errors.New("failed to reach target")
				}
				return nil
			}, cfg.Rc)(ctx)
			if err != nil {
				log.Debug("traceroute could not reach target", "ttl", ttl)
			}
		}(ttl)
	}

	wg.Wait()
	close(results)

	for r := range results {
		hops[r.Ttl] = append(hops[r.Ttl], r)
	}

	printHops(ctx, hops)

	return hops, nil
}

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

// traceroute performs a traceroute to the given address with the specified TTL and timeout.
// It returns a Hop struct containing the latency, TTL, address, and other details of the hop.
func traceroute(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (*Hop, error) {
	log := logger.FromContext(ctx)
	canIcmp, icmpListener, err := newIcmpListener()
	if err != nil {
		log.Error("Failed to open ICMP socket", "err", err.Error(), "ttl", ttl)
		return nil, err
	}
	defer closeIcmpListener(canIcmp, icmpListener)

	start := time.Now()
	conn, clientPort, err := tcpHop(addr, ttl, timeout)
	latency := time.Since(start)
	if err == nil {
		return handleTcpSuccess(conn, addr, ttl, latency), nil
	}

	if !canIcmp {
		log.Debug("No permission for icmp socket", "ttl", ttl)
		return &Hop{
			Latency: latency,
			Ttl:     ttl,
			Reached: false,
		}, nil
	}

	h := handleIcmpResponse(ctx, icmpListener, clientPort, ttl, timeout)
	h.Latency = latency
	return &h, nil
}

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

func closeIcmpListener(canIcmp bool, icmpListener *icmp.PacketConn) {
	if canIcmp && icmpListener != nil {
		icmpListener.Close() // #nosec G104
	}
}

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
		log.Debug("Reading ICMP message", "ttl", ttl)
		gotPort, addr, err := readIcmpMessage(ctx, icmpListener, timeout)
		if err != nil {
			log.Debug("Failed to read ICMP message", "err", err.Error(), "ttl", ttl)
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

	log.Debug("Deadline reached", "ttl", ttl)
	return Hop{
		Ttl: ttl,
	}
}

type Hop struct {
	Latency time.Duration `json:"latency" yaml:"latency" mapstructure:"latency"`
	Addr    HopAddress    `json:"addr" yaml:"addr" mapstructure:"addr"`
	Name    string        `json:"name" yaml:"name" mapstructure:"name"`
	Ttl     int           `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
	Reached bool          `json:"reached" yaml:"reached" mapstructure:"reached"`
}

type HopAddress struct {
	IP   string `json:"ip" yaml:"ip" mapstructure:"ip"`
	Port int    `json:"port" yaml:"port" mapstructure:"port"`
}

func (a HopAddress) String() string {
	if a.Port != 0 {
		return fmt.Sprintf("%s:%d", a.IP, a.Port)
	}
	return a.IP
}

func printHops(ctx context.Context, mapHops map[int][]Hop) {
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
			log.Debug(out)
		}
	}
}
