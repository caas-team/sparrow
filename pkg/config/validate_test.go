// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
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

package config

import (
	"context"
	"testing"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/test"
)

func TestConfig_Validate(t *testing.T) {
	test.MarkAsShort(t)

	ctx := context.Background()
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "config ok",
			config: Config{
				SparrowName: "sparrow.com",
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "https://test.de/config",
						Timeout: time.Second,
						RetryCfg: helper.RetryConfig{
							Count: 1,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},

			wantErr: false,
		},
		{
			name: "loader - url missing",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				SparrowName: "sparrow.com",
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "",
						Timeout: time.Second,
						RetryCfg: helper.RetryConfig{
							Count: 1,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},

			wantErr: true,
		},
		{
			name: "loader - url malformed",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				SparrowName: "sparrow.com",
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "this is not a valid url",
						Timeout: time.Second,
						RetryCfg: helper.RetryConfig{
							Count: 1,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "loader - retry count to high",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				SparrowName: "sparrow.com",
				Loader: LoaderConfig{
					Type: "http",
					Http: HttpLoaderConfig{
						Url:     "test.de",
						Timeout: time.Minute,
						RetryCfg: helper.RetryConfig{
							Count: 100000,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "loader - file path malformed",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				SparrowName: "sparrow.com",
				Loader: LoaderConfig{
					Type: "file",
					File: FileLoaderConfig{
						Path: "",
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "targetManager - Wrong Scheme",
			config: Config{
				Api: api.Config{
					ListeningAddress: ":8080",
				},
				SparrowName: "sparrow.com",
				Loader: LoaderConfig{
					Type: "file",
					File: FileLoaderConfig{
						Path: "",
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isDNSName(t *testing.T) {
	test.MarkAsShort(t)

	tests := []struct {
		name    string
		dnsName string
		want    bool
	}{
		{name: "dns name", dnsName: "sparrow.de", want: true},
		{name: "dns name with subdomain", dnsName: "sparrow.test.de", want: true},
		{name: "dns name with subdomain and tld and -", dnsName: "sub-sparrow.test.de", want: true},
		{name: "empty name", dnsName: "", want: false},
		{name: "dns name without tld", dnsName: "sparrow", want: false},
		{name: "name with underscore", dnsName: "test_de", want: false},
		{name: "name with space", dnsName: "test de", want: false},
		{name: "name with special chars", dnsName: "test!de", want: false},
		{name: "name with capitals", dnsName: "tEst.de", want: false},
		{name: "name with empty tld", dnsName: "tEst.de.", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDNSName(tt.dnsName); got != tt.want {
				t.Errorf("isDNSName() = %v, want %v", got, tt.want)
			}
		})
	}
}
