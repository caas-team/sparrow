package config

import (
	"testing"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
)

func TestConfig_Validate(t *testing.T) {
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
			if err := c.Validate(&RunFlagsNameMapping{}); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
