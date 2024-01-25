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

package logger

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name        string
		handlers    []slog.Handler
		expectedErr bool
		logLevelEnv string
	}{
		{
			name:        "No handler with default log level",
			handlers:    nil,
			expectedErr: false,
			logLevelEnv: "",
		},
		{
			name:        "No handler with DEBUG log level",
			handlers:    nil,
			expectedErr: false,
			logLevelEnv: "DEBUG",
		},
		{
			name:        "Custom handler provided",
			handlers:    []slog.Handler{slog.NewJSONHandler(os.Stdout, nil)},
			expectedErr: false,
			logLevelEnv: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.logLevelEnv)

			log := NewLogger(tt.handlers...)

			if (log == nil) != tt.expectedErr {
				t.Errorf("NewLogger() error = %v, expectedErr %v", log == nil, tt.expectedErr)
			}

			if tt.logLevelEnv != "" {
				want := getLevel(tt.logLevelEnv)
				got := log.Enabled(context.Background(), want)
				if !got {
					t.Errorf("Expected log level: %v", want)
				}
			}

			if len(tt.handlers) > 0 && !reflect.DeepEqual(log.Handler(), tt.handlers[0]) {
				t.Errorf("Handler not set correctly")
			}
		})
	}
}

func TestNewContextWithLogger(t *testing.T) {
	tests := []struct {
		name         string
		parentCtx    context.Context
		expectedType *slog.Logger
	}{
		{
			name:         "With Background context",
			parentCtx:    context.Background(),
			expectedType: (*slog.Logger)(nil),
		},
		{
			name:         "With already set logger in context",
			parentCtx:    context.WithValue(context.Background(), logger{}, NewLogger()),
			expectedType: (*slog.Logger)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := NewContextWithLogger(tt.parentCtx)
			defer cancel()

			log := ctx.Value(logger{})
			if _, ok := log.(*slog.Logger); !ok {
				t.Errorf("Context does not contain *slog.Logger, got %T", log)
			}
			if ctx == tt.parentCtx {
				t.Errorf("NewContextWithLogger returned the same context as the parent")
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		expect *slog.Logger
	}{
		{
			name:   "Context with logger",
			ctx:    IntoContext(context.Background(), NewLogger(slog.NewJSONHandler(os.Stdout, nil))),
			expect: NewLogger(slog.NewJSONHandler(os.Stdout, nil)),
		},
		{
			name:   "Context without logger",
			ctx:    context.Background(),
			expect: NewLogger(),
		},
		{
			name:   "Nil context",
			ctx:    nil,
			expect: NewLogger(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromContext(tt.ctx)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("FromContext() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		parentCtx   context.Context
		expectInCtx bool
	}{
		{
			name:        "With logger in parent context",
			parentCtx:   IntoContext(context.Background(), NewLogger()),
			expectInCtx: true,
		},
		{
			name:        "Without logger in parent context",
			parentCtx:   context.Background(),
			expectInCtx: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := Middleware(tt.parentCtx)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, ok := r.Context().Value(logger{}).(*slog.Logger)
				if tt.expectInCtx != ok {
					t.Errorf("Middleware() did not inject logger correctly, got %v, want %v", ok, tt.expectInCtx)
				}
			})

			req := httptest.NewRequest("GET", "/", http.NoBody)
			w := httptest.NewRecorder()

			middleware(handler).ServeHTTP(w, req)
		})
	}
}

func TestGetLevel(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect slog.Level
	}{
		{"Empty string", "", slog.LevelInfo},
		{"Debug level", "DEBUG", slog.LevelDebug},
		{"Info level", "INFO", slog.LevelInfo},
		{"Warn level", "WARN", slog.LevelWarn},
		{"Warning level", "WARNING", slog.LevelWarn},
		{"Error level", "ERROR", slog.LevelError},
		{"Invalid level", "UNKNOWN", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getLevel(tt.input)
			if got != tt.expect {
				t.Errorf("getLevel(%s) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}
