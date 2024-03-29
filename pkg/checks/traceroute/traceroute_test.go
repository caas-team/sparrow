package traceroute

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/aeden/traceroute"
	"github.com/caas-team/sparrow/pkg/checks"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		name string
		c    *Traceroute
		want map[string]result
	}{
		{
			name: "Success 5 hops",
			c:    newForTest(success(5), []string{"8.8.8.8"}),
			want: map[string]result{
				"8.8.8.8": {
					NumHops: 5,
					Hops: []hop{
						{Addr: "0.0.0.0", Latency: 0 * time.Second, Success: false},
						{Addr: "0.0.0.1", Latency: 1 * time.Second, Success: false},
						{Addr: "0.0.0.2", Latency: 2 * time.Second, Success: false},
						{Addr: "0.0.0.3", Latency: 3 * time.Second, Success: false},
						{Addr: "google-public-dns-a.google.com", Latency: 69 * time.Second, Success: true},
					},
				},
			},
		},
		{
			name: "Traceroute internal error fails silently",
			c:    newForTest(returnError(&net.DNSError{Err: "no such host", Name: "google.com", IsNotFound: true}), []string{"google.com"}),
			want: map[string]result{
				"google.com": {Hops: []hop{}},
			},
		},
	}

	for _, c := range cases {
		res := c.c.check(context.Background())

		if !cmp.Equal(res, c.want) {
			diff := cmp.Diff(res, c.want)
			t.Errorf("unexpected result: +want -got\n%s", diff)
		}
	}
}

func newForTest(f tracerouteFactory, targets []string) *Traceroute {
	t := make([]Target, len(targets))
	for i, target := range targets {
		t[i] = Target{Addr: target}
	}
	return &Traceroute{
		config: Config{
			Targets: t,
		},
		traceroute: f,
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}),
		},
	}
}

// success produces a tracerouteFactory that returns a traceroute result with nHops hops
func success(nHops int) tracerouteFactory {
	return func(dest string, port, timeout, retries, maxHops int) (traceroute.TracerouteResult, error) {
		hops := make([]traceroute.TracerouteHop, nHops)
		for i := 0; i < nHops-1; i++ {
			hops[i] = traceroute.TracerouteHop{
				Success:     false,
				N:           nHops,
				Host:        ipFromInt(i),
				ElapsedTime: time.Duration(i) * time.Second,
				TTL:         i,
			}
		}
		hops[nHops-1] = traceroute.TracerouteHop{
			Success:     true,
			Address:     [4]byte{8, 8, 8, 8},
			N:           nHops,
			Host:        "google-public-dns-a.google.com",
			ElapsedTime: 69 * time.Second,
			TTL:         nHops,
		}

		return traceroute.TracerouteResult{
			DestinationAddress: hops[nHops-1].Address,
			Hops:               hops,
		}, nil
	}
}

func returnError(err error) tracerouteFactory {
	return func(dest string, port, timeout, retries, maxHops int) (traceroute.TracerouteResult, error) {
		return traceroute.TracerouteResult{}, err
	}
}

// ipFromInt takes in an int and builds an IP address from it
// Example:
// ipFromInt(300) -> 0.0.1.44
func ipFromInt(i int) string {
	b1 := i >> 24 & 0xFF
	b2 := i >> 16 & 0xFF
	b3 := i >> 8 & 0xFF
	b4 := i & 0xFF

	return net.IPv4(byte(b1), byte(b2), byte(b3), byte(b4)).String()
}

func TestIpFromInt(t *testing.T) {
	cases := []struct {
		In       int
		Expected string
	}{
		{In: 300, Expected: "0.0.1.44"},
		{In: 0, Expected: "0.0.0.0"},
		{In: (1 << 33) - 1, Expected: "255.255.255.255"},
	}

	for _, c := range cases {
		t.Run("ipFromInt", func(t *testing.T) {
			actual := ipFromInt(c.In)
			if c.Expected != actual {
				t.Errorf("expected: %v, actual: %v", c.Expected, actual)
			}
		})
	}
}
