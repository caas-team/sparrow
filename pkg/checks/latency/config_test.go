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

package latency

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Targets:  []string{"http://localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid targets - invalid url",
			config: Config{
				Targets:  []string{"://localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid targets - invalid scheme",
			config: Config{
				Targets:  []string{"localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid interval",
			config: Config{
				Targets:  []string{"http://localhost:8080"},
				Interval: 10 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: Config{
				Targets:  []string{"http://localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  100 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
