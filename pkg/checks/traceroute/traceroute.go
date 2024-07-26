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
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// randomPort returns a random port in the interval [ 30_000, 40_000 [
func randomPort() int {
	return rand.Intn(10_000) + 30_000
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
			Control: func(network, address string, c syscall.RawConn) error {
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
		if !errors.Is(err, syscall.Errno(syscall.EADDRINUSE)) {
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
	icmpListener.SetReadDeadline(time.Now().Add(timeout))
	buffer := make([]byte, 1500) // Standard MTU size
	n, routerAddr, err := icmpListener.ReadFrom(buffer)
	if err != nil {
		// we probably timed out so return
		return 0, nil, fmt.Errorf("Failed to read from icmp connection: %w", err)
	}

	// Parse the ICMP message
	msg, err := icmp.ParseMessage(ipv4.ICMPTypeTimeExceeded.Protocol(), buffer[:n])
	if err != nil {
		return 0, nil, err
	}

	// Ensure the message is an ICMP Time Exceeded message
	if msg.Type != ipv4.ICMPTypeTimeExceeded {
		return 0, nil, errors.New("Message is not 'Time Exceeded'")
	}

	// The first 20 bytes of Data are the IP header, so the TCP segment starts at byte 20
	tcpSegment := msg.Body.(*icmp.TimeExceeded).Data[20:]

	// Extract the source port from the TCP segment
	destPort := int(tcpSegment[0])<<8 + int(tcpSegment[1])

	return destPort, routerAddr, nil
}

// TraceRoute performs a traceroute to the specified host using TCP and listens for ICMP Time Exceeded messages using ICMP.
func TraceRoute(host string, port, timeout, maxHops int, rc helper.RetryConfig) (map[int][]Hop, error) {
	// this could also be a 2d array, but I feel like using an int map here makes the json easier to understand
	// as it explicitly shows a mapping of ttl->hops
	var hops map[int][]Hop

	toDuration := time.Duration(timeout) * time.Second

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	results := make(chan Hop, maxHops)
	var wg sync.WaitGroup

	for ttl := 1; ttl <= maxHops; ttl++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			helper.Retry(func(ctx context.Context) error {
				reached, err := traceroute(results, addr, ttl, toDuration, rc)
				if err != nil {
					return err
				}
				if !reached {
					return errors.New("failed to reach target, please retry")
				}
				return nil
			}, rc)
		}()
	}

	wg.Wait()
	close(results)

	for r := range results {
		hops[r.Ttl] = append(hops[r.Ttl], r)
	}

	// TODO: log this on debug level
	printHops(hops)

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

func traceroute(results chan Hop, addr net.Addr, ttl int, timeout time.Duration, rc helper.RetryConfig) (bool, error) {
	icmpListener, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false, err
	}
	defer icmpListener.Close()
	start := time.Now()
	conn, clientPort, err := tcpHop(addr, ttl, timeout)
	latency := time.Since(start)
	if err == nil {
		conn.Close()

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
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Unix() < deadline.Unix() && !found {
		gotPort, addr, err := readIcmpMessage(icmpListener, timeout)
		if err != nil {
			results <- Hop{
				Latency: latency,
				Ttl:     ttl,
				Reached: false,
			}
			return false, nil
		}

		// Check if the destination port matches our dialer's source port
		if gotPort == clientPort {
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

func printHops(mapHops map[int][]Hop) {
	for ttl, hops := range mapHops {
		for _, hop := range hops {
			fmt.Printf("%d %s %s %v ", ttl, hop.Addr, hop.Name, hop.Latency)
			if hop.Reached {
				fmt.Print("( Reached )")
			}
			fmt.Println()
		}
	}
}
