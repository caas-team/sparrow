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

package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

type logger struct{}

// NewLogger creates a new slog.Logger instance.
// If handlers are provided, the first handler in the slice is used; otherwise,
// a default JSON handler writing to os.Stderr is used. This function allows for
// custom configuration of logging handlers.
func NewLogger(h ...slog.Handler) *slog.Logger {
	var handler slog.Handler
	if len(h) > 0 {
		handler = h[0]
	} else {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     getLevel(os.Getenv("LOG_LEVEL")),
		})
	}
	return slog.New(handler)
}

// NewContextWithLogger creates a new context based on the provided parent context.
// It embeds a logger into this new context.
// It also returns a cancel function to cancel the new context.
func NewContextWithLogger(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	return IntoContext(ctx, FromContext(parent)), cancel
}

// IntoContext embeds the provided slog.Logger into the given context and returns the modified context.
// This function is used for passing loggers through context, allowing for context-aware logging.
func IntoContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, logger{}, log)
}

// FromContext extracts the slog.Logger from the provided context.
// If the context does not have a logger, it returns a new logger with the default configuration.
// This function is useful for retrieving loggers from context in different parts of an application.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(logger{}).(*slog.Logger); ok {
			return logger
		}
	}
	return NewLogger()
}

// Middleware takes the logger from the context and adds it to the request context
func Middleware(ctx context.Context) func(http.Handler) http.Handler {
	log := FromContext(ctx)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCtx := IntoContext(r.Context(), log)
			next.ServeHTTP(w, r.WithContext(reqCtx))
		})
	}
}

// getLevel takes a level string and maps it to the corresponding slog.Level
// Returns the level if no mapped level is found it returns info level
func getLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
