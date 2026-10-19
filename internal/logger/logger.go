package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Level  string
	Format string
}

type contextKey struct{}

var requestIDKey = contextKey{}

func New(cfg Config) *slog.Logger {
	level := slog.LevelInfo

	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(&ContextHandler{
		Handler: handler,
	})
}

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(
	ctx context.Context,
	record slog.Record,
) error {
	if ctx != nil {
		if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
			record.AddAttrs(slog.String("request_id", requestID))
		}

		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			record.AddAttrs(
				slog.String("trace_id", span.SpanContext().TraceID().String()),
				slog.String("span_id", span.SpanContext().SpanID().String()),
			)
		}
	}

	return h.Handler.Handle(ctx, record)
}

func WithRequestID(
	ctx context.Context,
	requestID string,
) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}

	return ""
}
