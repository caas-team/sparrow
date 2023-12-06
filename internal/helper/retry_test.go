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

package helper

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	effectorFuncCallCounter := 0
	ctx, cancel := context.WithCancel(context.Background())

	type args struct {
		effector Effector
		rc       RetryConfig
	}
	tests := []struct {
		name        string
		args        args
		ctx         context.Context
		wantRetries int
		wantError   bool
	}{
		{
			name: "success after first call",
			args: args{
				effector: func(ctx context.Context) error {
					effectorFuncCallCounter++
					return nil
				},
				rc: RetryConfig{
					Count: 2,
					Delay: time.Second,
				},
			},
			ctx:         context.Background(),
			wantError:   false,
			wantRetries: 0,
		},
		{
			name: "success after first retry",
			args: args{
				effector: func(ctx context.Context) error {
					effectorFuncCallCounter++
					if effectorFuncCallCounter > 1 {
						return nil
					}
					return fmt.Errorf("Ups sth wrong")
				},
				rc: RetryConfig{
					Count: 2,
					Delay: time.Second,
				},
			},
			ctx:         context.Background(),
			wantError:   false,
			wantRetries: 1,
		},
		{
			name: "error",
			args: args{
				effector: func(ctx context.Context) error {
					effectorFuncCallCounter++
					return fmt.Errorf("Ups sth wrong")
				},
				rc: RetryConfig{
					Count: 2,
					Delay: time.Second,
				},
			},
			ctx:         context.Background(),
			wantError:   true,
			wantRetries: 2,
		},
		{
			name: "context timeout",
			args: args{
				effector: func(ctx context.Context) error {
					effectorFuncCallCounter++
					cancel()
					return errors.New("Ups")
				},
				rc: RetryConfig{
					Count: 2,
					Delay: time.Second,
				},
			},
			ctx:         ctx,
			wantError:   true,
			wantRetries: 0,
		},
	}
	for _, tt := range tests {
		effectorFuncCallCounter = 0
		t.Run(tt.name, func(t *testing.T) {
			retry := Retry(tt.args.effector, tt.args.rc)
			err := retry(tt.ctx)
			if (err != nil) != tt.wantError {
				t.Errorf("Retry() error = %v, wantErr %v", err, tt.wantError)
				return
			}
			if effectorFuncCallCounter-1 != tt.wantRetries {
				t.Errorf("Retry() gotReties = %v, want %v", effectorFuncCallCounter-1, tt.wantRetries)
			}
		})
	}
}

func Test_getExpBackoff(t *testing.T) {
	type args struct {
		initialDelay time.Duration
		iteration    int
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "1 sec and 1. iteration",
			args: args{
				initialDelay: time.Second,
				iteration:    1,
			},
			want: time.Second,
		},
		{
			name: "1 sec and 2. iteration",
			args: args{
				initialDelay: time.Second,
				iteration:    2,
			},
			want: time.Second * 2,
		},
		{
			name: "1 sec and 3. iteration",
			args: args{
				initialDelay: time.Second,
				iteration:    3,
			},
			want: time.Second * 4,
		},
		{
			name: "1 sec and 4. iteration",
			args: args{
				initialDelay: time.Second,
				iteration:    4,
			},
			want: time.Second * 8,
		},
		{
			name: "1 sec and unknown iteration",
			args: args{
				initialDelay: time.Second,
				iteration:    -12,
			},
			want: time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getExpBackoff(tt.args.initialDelay, tt.args.iteration); got != tt.want {
				t.Errorf("getExpBackoff() = %v, want %v", got, tt.want)
			}
		})
	}
}
