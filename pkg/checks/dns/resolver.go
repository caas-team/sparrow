package dns

import (
	"context"
	"net"
)

//go:generate moq -out resolver_moq.go . Resolver
type Resolver interface {
	LookupAddr(ctx context.Context, addr string) ([]string, error)
	LookupHost(ctx context.Context, addr string) ([]string, error)
	SetDialer(d *net.Dialer)
}

type resolver struct {
	*net.Resolver
}

func NewResolver() Resolver {
	return &resolver{
		Resolver: &net.Resolver{},
	}
}

func (r *resolver) SetDialer(d *net.Dialer) {
	r.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		return d.DialContext(ctx, network, address)
	}
}
