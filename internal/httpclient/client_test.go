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
	"reflect"
	"testing"
)

func TestIntoContext(t *testing.T) {
	mockClient := &http.Client{}

	tests := []struct {
		name    string
		client  *http.Client
		wantNil bool
	}{
		{
			name:    "nil client",
			client:  nil,
			wantNil: true,
		},
		{
			name:    "valid client",
			client:  mockClient,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := IntoContext(context.Background(), tt.client)
			if ctx == nil {
				t.Fatal("IntoContext returned a nil context")
			}

			c, ok := ctx.Value(client{}).(*http.Client)
			if !ok && !tt.wantNil {
				t.Errorf("Expected a client, got none")
			}

			if !reflect.DeepEqual(c, tt.client) {
				t.Errorf("Client got = %v, want %v", c, tt.client)
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	mockClient := &http.Client{}

	tests := []struct {
		name      string
		ctxClient *http.Client
		want      *http.Client
	}{
		{
			name:      "no client in context",
			ctxClient: nil,
			want:      http.DefaultClient,
		},
		{
			name:      "client in context",
			ctxClient: mockClient,
			want:      mockClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.ctxClient != nil {
				ctx = IntoContext(ctx, tt.ctxClient)
			}

			if got := FromContext(ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
