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
	type want struct {
		expected map[string]Result
		wantErr  bool
		err      error
	}
	type testcase struct {
		name string
		c    *Traceroute
		want want
	}

	cases := []testcase{
		{
			name: "Success 5 hops",
			c:    newForTest(success(5), []string{"8.8.8.8"}),
			want: want{
				expected: map[string]Result{
					"8.8.8.8": {
						NumHops: 5,
						Hops: []Hop{
							{Addr: "0.0.0.0", Latency: 0 * time.Second, Success: false},
							{Addr: "0.0.0.1", Latency: 1 * time.Second, Success: false},
							{Addr: "0.0.0.2", Latency: 2 * time.Second, Success: false},
							{Addr: "0.0.0.3", Latency: 3 * time.Second, Success: false},
							{Addr: "google-public-dns-a.google.com", Latency: 69 * time.Second, Success: true},
						},
					},
				},
				wantErr: false,
			},
		},
		{
			name: "Traceroute internal error",
			c:    newForTest(returnError(&net.DNSError{Err: "no such host", Name: "google.com", IsNotFound: true}), []string{"google.com"}),
			want: want{
				wantErr: true,
				expected: map[string]Result{
					"google.com": {Hops: []Hop{}},
				},
				err: &net.DNSError{Err: "no such host", Name: "google.com", IsNotFound: true},
			},
		},
	}

	for _, c := range cases {
		res, err := c.c.check(context.Background())

		if c.want.wantErr {
			if err == nil {
				t.Errorf("expected error, got nil")
			} else if c.want.err.Error() != err.Error() {
				t.Errorf("expected: %v, got: %v", c.want.err, err)
			}
		}
		if !cmp.Equal(res, c.want.expected) {
			diff := cmp.Diff(res, c.want.expected)
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
			Mu:      sync.Mutex{},
			CResult: make(chan checks.Result),
			Done:    make(chan bool),
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
	type testcase struct {
		In       int
		Expected string
	}
	cases := []testcase{
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
