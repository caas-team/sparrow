package targets

import (
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
				Config: Config{
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
				},
			},
		},
		{
			name: "valid config - zero values",
			cfg: TargetManagerConfig{
				Config: Config{
					UnhealthyThreshold:   0,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 0,
				},
			},
		},
		{
			name: "invalid config - zero check interval",
			cfg: TargetManagerConfig{
				Config: Config{
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        0,
					RegistrationInterval: 1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - negative values",
			cfg: TargetManagerConfig{
				Config: Config{
					UnhealthyThreshold:   -1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
