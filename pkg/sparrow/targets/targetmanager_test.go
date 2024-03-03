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

package targets

import (
	"context"
	"testing"
	"time"
)

func TestTargetManagerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     TargetManagerConfig
		wantErr bool
	}{
		{
			name:    "empty config",
			wantErr: true,
		},
		{
			name: "valid config - non-zero values",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
		},
		{
			name: "valid config - zero values",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					UnhealthyThreshold:   0,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 0,
					UpdateInterval:       0,
				},
			},
		},
		{
			name: "invalid config - zero check interval",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        0,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - negative values",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					UnhealthyThreshold:   -1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - unknown interactor",
			cfg: TargetManagerConfig{
				Type: "unknown",
				General: General{
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
