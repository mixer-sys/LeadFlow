package logging

import (
	"context"
	"log/slog"
	"os"
)

type Config struct {
	Level  string
	Format string
}

func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	options := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, options)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}

	logger := slog.New(handler)

	return logger.With(
		slog.String("service", "leadflow"),
	)
}

func parseLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type requestIDKey struct{}

func WithRequestID(
	ctx context.Context,
	requestID string,
) context.Context {
	return context.WithValue(ctx, requestIDKey{},
		requestID)
}

func RequestID(ctx context.Context) string {
	if value, ok := ctx.Value(requestIDKey{}).(string); ok {
		return value
	}

	return ""
}
