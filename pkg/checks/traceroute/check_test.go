package traceroute

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

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
					Hops: []Hop{
						{Addr: &net.TCPAddr{IP: net.ParseIP("0.0.0.0")}, Latency: 0 * time.Second, Reached: false, Ttl: 1},
						{Addr: &net.TCPAddr{IP: net.ParseIP("0.0.0.1")}, Latency: 1 * time.Second, Reached: false, Ttl: 2},
						{Addr: &net.TCPAddr{IP: net.ParseIP("0.0.0.2")}, Latency: 2 * time.Second, Reached: false, Ttl: 3},
						{Addr: &net.TCPAddr{IP: net.ParseIP("0.0.0.3")}, Latency: 3 * time.Second, Reached: false, Ttl: 4},
						{Addr: &net.TCPAddr{IP: net.ParseIP("123.0.0.123"), Port: 53}, Name: "google-public-dns-a.google.com", Latency: 69 * time.Second, Reached: true, Ttl: 5},
					},
				},
			},
		},
		{
			name: "Traceroute internal error fails silently",
			c:    newForTest(returnError(&net.DNSError{Err: "no such host", Name: "google.com", IsNotFound: true}), []string{"google.com"}),
			want: map[string]result{
				"google.com": {Hops: []Hop{}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := c.c.check(context.Background())

			if !cmp.Equal(res, c.want) {
				diff := cmp.Diff(res, c.want)
				t.Errorf("unexpected result: +want -got\n%s", diff)
			}
		})
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
	return func(dest string, port, timeout, retries, maxHops int) ([]Hop, error) {
		hops := make([]Hop, nHops)
		for i := 0; i < nHops-1; i++ {
			hops[i] = Hop{
				Latency: time.Second * time.Duration(i),
				Addr:    &net.TCPAddr{IP: net.ParseIP(ipFromInt(i))},
				Name:    "",
				Ttl:     i + 1,
				Reached: false,
			}
		}
		hops[nHops-1] = Hop{
			Latency: 69 * time.Second,
			Addr: &net.TCPAddr{
				IP:   net.ParseIP("123.0.0.123"),
				Port: 53,
			},
			Name:    "google-public-dns-a.google.com",
			Ttl:     nHops,
			Reached: true,
		}

		return hops, nil
	}
}

func returnError(err error) tracerouteFactory {
	return func(dest string, port, timeout, retries, maxHops int) ([]Hop, error) {
		return []Hop{}, err
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
