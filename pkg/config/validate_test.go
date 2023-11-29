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
	"github.com/caas-team/sparrow/internal/logger"
)

func TestConfig_Validate(t *testing.T) {
	ctx, cancel := logger.NewContextWithLogger(context.Background(), "validation-test")
	defer cancel()

	type fields struct {
		Loader LoaderConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "config ok",
			fields: fields{
				Loader: LoaderConfig{
					Type: "http",
					http: HttpLoaderConfig{
						url:     "https://test.de/config",
						timeout: time.Second,
						retryCfg: helper.RetryConfig{
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
			name: "url missing",
			fields: fields{
				Loader: LoaderConfig{
					Type: "http",
					http: HttpLoaderConfig{
						url:     "",
						timeout: time.Second,
						retryCfg: helper.RetryConfig{
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
			name: "url malformed",
			fields: fields{
				Loader: LoaderConfig{
					Type: "http",
					http: HttpLoaderConfig{
						url:     "this is not a valid url",
						timeout: time.Second,
						retryCfg: helper.RetryConfig{
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
			name: "retry count to high",
			fields: fields{
				Loader: LoaderConfig{
					Type: "http",
					http: HttpLoaderConfig{
						url:     "test.de",
						timeout: time.Minute,
						retryCfg: helper.RetryConfig{
							Count: 100000,
							Delay: time.Second,
						},
					},
					Interval: time.Second,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Checks: nil,
				Loader: tt.fields.Loader,
			}
			if err := c.Validate(ctx, &RunFlagsNameMapping{}); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
