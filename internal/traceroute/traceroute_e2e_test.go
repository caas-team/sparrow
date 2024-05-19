package traceroute_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/caas-team/sparrow/internal/traceroute"
)

func TestTracer_Run_E2E(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tests := []struct {
		name    string
		address string
		maxHops int
		timeout time.Duration
		wantErr bool
		wantIPs []net.IP
	}{
		{
			name:    "IPv4 Google",
			address: "google.com",
			maxHops: 30,
			timeout: 2 * time.Second,
			wantErr: false,
			wantIPs: lookupIP(t, "google.com"),
		},
		{
			name:    "IPv6 Google",
			address: "google.com",
			maxHops: 30,
			timeout: 2 * time.Second,
			wantErr: false,
			wantIPs: lookupIP(t, "google.com"),
		},
		{
			name:    "Invalid address",
			address: "invalid.address",
			maxHops: 30,
			timeout: 2 * time.Second,
			wantErr: true,
		},
		{
			name:    "IPv4 Localhost",
			address: "localhost",
			maxHops: 30,
			timeout: 2 * time.Second,
			wantErr: false,
			wantIPs: []net.IP{net.ParseIP("127.0.0.1")},
		},
		{
			name:    "IPv6 Localhost",
			address: "::1",
			maxHops: 30,
			timeout: 2 * time.Second,
			wantErr: false,
			wantIPs: []net.IP{net.ParseIP("::1")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := traceroute.New(tt.maxHops, tt.timeout, traceroute.ICMP)
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			hops, err := tracer.Run(ctx, tt.address)

			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
				return
			}

			if !tt.wantErr {
				if len(hops) == 0 {
					t.Errorf("expected at least one hop, got none")
				}

				if len(tt.wantIPs) > 0 {
					if !containsIP(hops, tt.wantIPs) {
						t.Errorf("expected IP addresses: %v, got: %v", tt.wantIPs, hops)
					}
				}
			}
		})
	}
}

// lookupIP resolves the given address and returns the list of IP addresses.
func lookupIP(t *testing.T, address string) []net.IP {
	t.Helper()

	ips, err := net.LookupIP(address)
	if err != nil {
		t.Fatalf("error resolving IP address: %v", err)
	}

	return ips
}

// containsIP checks if the given list of hops contains any of the given IP addresses.
func containsIP(hops []traceroute.Hop, ips []net.IP) bool {
	for _, ip := range ips {
		for _, hop := range hops {
			if hop.IP.Equal(ip) {
				return true
			}
		}
	}

	return false
}
