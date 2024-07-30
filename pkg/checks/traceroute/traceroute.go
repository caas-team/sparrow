package traceroute

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/internal/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
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
					opErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
				}); err != nil {
					return err
				}
				return opErr
			},
		}

		// Attempt to connect to the target host
		conn, err := dialer.Dial("tcp", addr.String())
		if !errors.Is(err, syscall.EADDRINUSE) {
			return conn, port, err
		}
	}
}

// readIcmpMessage reads a packet from the provided icmp Connection. If the packet is 'Time Exceeded',
// it reads the address of the router that dropped created the icmp packet. It also reads the source port
// from the payload and finds the source port used by the previous tcp connection. If any error is returned,
// an icmp packet was either not received, or the received packet was not a time exceeded.
func readIcmpMessage(icmpListener *icmp.PacketConn, timeout time.Duration) (int, net.Addr, error) {
	// Expected to fail due to TTL expiry, listen for ICMP response
	if err := icmpListener.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return 0, nil, fmt.Errorf("failed to set icmp read deadline: %w", err)
	}
	buffer := make([]byte, 1500) //nolint:mnd // Standard MTU size
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
	if msg.Type != ipv4.ICMPTypeTimeExceeded {
		return 0, nil, errors.New("message is not 'Time Exceeded'")
	}

	// The first 20 bytes of Data are the IP header, so the TCP segment starts at byte 20
	tcpSegment := msg.Body.(*icmp.TimeExceeded).Data[20:]

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
				reached, err := traceroute(results, addr, ttl, timeoutDuration)
				if err != nil {
					log.Error("traceroute failed", "err", err.Error(), "ttl", ttl)
					return err
				}
				if !reached {
					log.Debug("failed to reach target, retrying", "ttl", ttl)
					return errors.New("failed to reach target")
				}
				return nil
			}, cfg.Rc)(ctx)
			if err != nil {
				log.Error("traceroute could not reach target", "ttl", ttl)
			}
		}(ttl)
	}

	wg.Wait()
	close(results)

	for r := range results {
		hops[r.Ttl] = append(hops[r.Ttl], r)
	}

	log.Debug("finished trace", "hops", printHops(hops))

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

func traceroute(results chan Hop, addr net.Addr, ttl int, timeout time.Duration) (bool, error) {
	canIcmp := true
	icmpListener, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		if !errors.Is(err, syscall.EPERM) {
			return false, err
		}
		canIcmp = false
	}

	defer func() {
		if canIcmp {
			icmpListener.Close() // #nosec G104
		}
	}()
	start := time.Now()
	conn, clientPort, err := tcpHop(addr, ttl, timeout)
	latency := time.Since(start)
	if err == nil {
		conn.Close() // #nosec G104

		ipaddr := ipFromAddr(addr)
		names, _ := net.LookupAddr(ipaddr.String()) // we don't care about this lookup failling

		name := ""
		if len(names) >= 1 {
			name = names[0]
		}

		results <- Hop{
			Latency: latency,
			Ttl:     ttl,
			Addr:    addr,
			Name:    name,
			Reached: true,
		}
		return true, nil
	}

	found := false
	deadline := time.Now().Add(timeout)

	for time.Now().Unix() < deadline.Unix() && !found {
		gotPort := -1
		var addr net.Addr
		if canIcmp {
			gotPort, addr, err = readIcmpMessage(icmpListener, timeout)
			if err != nil {
				results <- Hop{
					Latency: latency,
					Ttl:     ttl,
					Reached: false,
				}
				return false, nil
			}
		}

		// Check if the destination port matches our dialer's source port
		if canIcmp && gotPort == clientPort {
			ipaddr := ipFromAddr(addr)
			names, _ := net.LookupAddr(ipaddr.String()) // we don't really care if this lookup works, so ignore the error

			name := ""
			if len(names) >= 1 {
				name = names[0]
			}

			results <- Hop{
				Latency: latency,
				Ttl:     ttl,
				Addr:    addr,
				Reached: false,
				Name:    name,
			}
			found = true
			break
		}
	}
	if !found {
		results <- Hop{
			Latency: latency,
			Ttl:     ttl,
			Reached: false,
		}
	}

	return false, nil
}

type Hop struct {
	Latency time.Duration
	Addr    net.Addr
	Name    string
	Ttl     int
	Reached bool
}

func printHops(mapHops map[int][]Hop) string {
	out := ""
	for ttl, hops := range mapHops {
		for _, hop := range hops {
			out += fmt.Sprintf("%d %s %s %v ", ttl, hop.Addr, hop.Name, hop.Latency)
			if hop.Reached {
				out += "( Reached )"
			}
			out += "\n"
		}
	}

	return out
}
