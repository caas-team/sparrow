package logger

import (
	"context"
	"log/slog"
	"os"
)

type logger struct{}

func GetLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, nil))
}

func IntoContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, logger{}, log)
}

func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(logger{}).(*slog.Logger); ok {
			return logger
		}
	}
	return GetLogger()
}
