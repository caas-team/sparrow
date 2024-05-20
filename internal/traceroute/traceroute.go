package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

var _ Tracer = (*tracer)(nil)

// Protocol defines the protocol used for the traceroute
type Protocol int

const (
	// ICMP represents the ICMP protocol
	ICMP Protocol = iota
	// UDP represents the UDP protocol
	UDP
	// TCP represents the TCP protocol
	TCP
	// bufferSize represents the buffer size for the received data
	bufferSize = 1500
)

// MaxConcurrentHops represents the maximum number of concurrent hops to perform
var MaxConcurrentHops = 10

// Tracer represents a traceroute implementation
//
//go:generate moq -out traceroute_moq.go . Tracer
type Tracer interface {
	// Run performs a traceroute to the given address using the specified protocol
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
	Duration float64
	// ReachedTarget indicates whether the target was reached with this hop
	ReachedTarget bool
}

// Run performs a traceroute to the given address using the specified protocol
func (t *tracer) Run(ctx context.Context, address string, port uint16) ([]Hop, error) {
	destAddr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return nil, fmt.Errorf("error resolving IP address: %w", err)
	}

	hopCh := make(chan Hop, t.MaxHops)
	errCh := make(chan error, t.MaxHops)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	reached := false
	p := &performer{
		wg:            &sync.WaitGroup{},
		mu:            &sync.Mutex{},
		sem:           make(chan struct{}, MaxConcurrentHops),
		hopCh:         hopCh,
		errCh:         errCh,
		reachedTarget: &reached,
		cancel:        cancel,
	}

	for ttl := 1; ttl <= t.MaxHops; ttl++ {
		p.wg.Add(1)
		go p.hop(ctx, destAddr, port, ttl, t.hop)
	}

	p.wg.Wait()
	close(hopCh)
	close(errCh)

	return t.processResults(hopCh, errCh)
}

// performer represents a performer of the traceroute
type performer struct {
	// wg is the WaitGroup for the performer
	wg *sync.WaitGroup
	// mu is the Mutex for the performer
	mu *sync.Mutex
	// sem is the semaphore to limit the number of concurrent hops
	sem chan struct{}
	// hopCh is the channel to send the hops
	hopCh chan<- Hop
	// errCh is the channel to send the errors
	errCh chan<- error
	// reachedTarget is a pointer to a boolean indicating whether the target was reached
	reachedTarget *bool
	// cancel is the cancel function for the context to stop the traceroute
	cancel context.CancelFunc
}

// hopperFunc represents a function that performs a hop in the traceroute
type hopperFunc func(context.Context, *net.IPAddr, uint16, int) (Hop, error)

// hop performs a single hop in the traceroute to the given address with the specified TTL
func (p *performer) hop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int, hopper hopperFunc) {
	defer p.wg.Done()

	select {
	case <-ctx.Done():
		return
	case p.sem <- struct{}{}:
	}
	defer func() { <-p.sem }()

	hop, err := hopper(ctx, destAddr, port, ttl)
	if err != nil {
		hop.IP = destAddr.IP
		hop.ReachedTarget = false
		hop.Error = err.Error()
		p.errCh <- err
	}
	p.hopCh <- hop

	if hop.ReachedTarget {
		p.mu.Lock()
		if !*p.reachedTarget {
			*p.reachedTarget = true
			p.cancel()
		}
		p.mu.Unlock()
	}
}

// processResults processes the results from the hop channel and error channel
func (t *tracer) processResults(hopCh <-chan Hop, errCh <-chan error) ([]Hop, error) {
	hops := t.collectHops(hopCh)
	filteredHops := t.filterHops(hops)

	for err := range errCh {
		if err != nil {
			return filteredHops, err
		}
	}
	return filteredHops, nil
}

// collectHops collects the hops from the hop channel
func (t *tracer) collectHops(hopCh <-chan Hop) []Hop {
	var hops []Hop
	for hop := range hopCh {
		hops = append(hops, hop)
	}
	sort.Slice(hops, func(i, j int) bool {
		return hops[i].Tracepoint < hops[j].Tracepoint
	})
	return hops
}

// filterHops filters the hops to remove the ones that are after the target has been reached
func (t *tracer) filterHops(hops []Hop) []Hop {
	var filtered []Hop
	reached := false
	for _, hop := range hops {
		if reached && hop.ReachedTarget {
			continue
		}
		if hop.ReachedTarget {
			reached = true
		}
		filtered = append(filtered, hop)
	}
	return filtered
}

// hop performs a single hop in the traceroute to the given address with the specified TTL.
func (t *tracer) hop(ctx context.Context, destAddr *net.IPAddr, port uint16, ttl int) (Hop, error) {
	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return Hop{
			Tracepoint: ttl,
			Error:      fmt.Sprintf("timeout after %fs", t.Timeout.Seconds()),
		}, ctx.Err()
	default:
		switch t.Protocol {
		case ICMP:
			return t.hopICMP(destAddr, ttl)
		case UDP:
			return Hop{}, errors.New("UDP not supported yet")
		case TCP:
			return t.hopTCP(ctx, destAddr, port, ttl)
		default:
			return Hop{}, errors.New("protocol not supported")
		}
	}
}
