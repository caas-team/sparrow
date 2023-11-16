package logger

import (
	"context"
	"log/slog"
	"os"
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
		handler = slog.NewJSONHandler(os.Stderr, nil)
	}
	return slog.New(handler)
}

// NewContextWithLogger creates a new context based on the provided parent context.
// It embeds a logger into this new context, which is a child of the logger from the parent context.
// The child logger inherits settings from the parent and is grouped under the provided childName.
// It also returns a cancel function to cancel the new context.
func NewContextWithLogger(parent context.Context, childName string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	return IntoContext(ctx, FromContext(parent).WithGroup(childName)), cancel
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
