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

package httpclient

import (
	"context"
	"net/http"

	"github.com/caas-team/sparrow/internal/logger"
)

type client struct{}

// IntoContext embeds the provided http.Client into the given context and returns the modified context.
// This function is used for passing http clients through context, allowing for easier request handling and client management.
func IntoContext(ctx context.Context, c *http.Client) context.Context {
	return context.WithValue(ctx, client{}, c)
}

// FromContext extracts the http.Client from the provided context.
// If the context does not have a client it returns http.DefaultClient.
// This function is useful for retrieving http clients from context in different parts of an application.
func FromContext(ctx context.Context) *http.Client {
	log := logger.FromContext(ctx)
	if ctx != nil {
		if c, ok := ctx.Value(client{}).(*http.Client); ok {
			return c
		}
	}

	log.Warn("No http.Client found in context; using http.DefaultClient")
	return http.DefaultClient
}
