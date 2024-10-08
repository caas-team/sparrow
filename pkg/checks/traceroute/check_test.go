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
	"net"
	"sync"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		name string
		c    *Traceroute
		want map[string]result
	}{
		{
			name: "Success 5 hops",
			c:    newForTest(success(5), 10, []string{"8.8.8.8"}),
			want: map[string]result{
				"8.8.8.8": {
					MinHops: 5,
					Hops: map[int][]Hop{
						1: {{Addr: HopAddress{IP: "0.0.0.1"}, Latency: 1 * time.Second, Reached: false, Ttl: 1}},
						2: {{Addr: HopAddress{IP: "0.0.0.2"}, Latency: 2 * time.Second, Reached: false, Ttl: 2}},
						3: {{Addr: HopAddress{IP: "0.0.0.3"}, Latency: 3 * time.Second, Reached: false, Ttl: 3}},
						4: {{Addr: HopAddress{IP: "0.0.0.4"}, Latency: 4 * time.Second, Reached: false, Ttl: 4}},
						5: {{Addr: HopAddress{IP: "123.0.0.123", Port: 53}, Name: "google-public-dns-a.google.com", Latency: 69 * time.Second, Reached: true, Ttl: 5}},
					},
				},
			},
		},
		{
			name: "Traceroute internal error fails silently",
			c:    newForTest(returnError(&net.DNSError{Err: "no such host", Name: "google.com", IsNotFound: true}), 10, []string{"google.com"}),
			want: map[string]result{
				"google.com": {MinHops: 10, Hops: map[int][]Hop{}},
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

func newForTest(f tracerouteFactory, maxHops int, targets []string) *Traceroute {
	t := make([]Target, len(targets))
	for i, target := range targets {
		t[i] = Target{Addr: target}
	}
	return &Traceroute{
		CheckBase:  checks.CheckBase{Mu: sync.Mutex{}, DoneChan: make(chan struct{})},
		config:     Config{Targets: t, MaxHops: maxHops},
		traceroute: f,
		metrics:    newMetrics(),
		tracer:     otel.Tracer("tracer.traceroute"),
	}
}

// success produces a tracerouteFactory that returns a traceroute result with nHops hops
func success(nHops int) tracerouteFactory {
	return func(ctx context.Context, cfg tracerouteConfig) (map[int][]Hop, error) {
		hops := make(map[int][]Hop)
		for i := 1; i < nHops; i++ {
			hops[i] = []Hop{
				{
					Latency: time.Second * time.Duration(i),
					Addr:    HopAddress{IP: ipFromInt(i)},
					Name:    "",
					Ttl:     i,
					Reached: false,
				},
			}
		}
		hops[nHops] = []Hop{
			{
				Latency: 69 * time.Second,
				Addr: HopAddress{
					IP:   "123.0.0.123",
					Port: 53,
				},
				Name:    "google-public-dns-a.google.com",
				Ttl:     nHops,
				Reached: true,
			},
		}

		return hops, nil
	}
}

func returnError(err error) tracerouteFactory {
	return func(_ context.Context, _ tracerouteConfig) (map[int][]Hop, error) {
		return map[int][]Hop{}, err
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
