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

func newResolver() Resolver {
	return &resolver{
		Resolver: &net.Resolver{
			// We need to set this so the custom dialer is used
			PreferGo: true,
		},
	}
}

func (r *resolver) SetDialer(d *net.Dialer) {
	r.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		return d.DialContext(ctx, network, address)
	}
}
