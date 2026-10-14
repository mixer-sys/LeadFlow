package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type requestIDKey struct{}

const requestIDHeader = "X-Request-ID"

type responseWriter struct {
	http.ResponseWriter

	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func RequestLogging(
	logger *slog.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := r.Header.Get(requestIDHeader)

			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set(requestIDHeader, requestID)

			ctx := context.WithValue(
				r.Context(),
				requestIDKey{},
				requestID,
			)

			r = r.WithContext(ctx)

			recorder := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			logger.Info(
				"http request completed",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", recorder.status),
				slog.Float64(
					"duration_ms",
					float64(time.Since(start).Microseconds())/1000.0,
				),
			)
		})
	}
}

func GetRequestID(r *http.Request) string {
	if requestID, ok := r.Context().Value(requestIDKey{}).(string); ok {
		return requestID
	}

	return ""
}

func generateRequestID() string {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return time.Now().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)
}

func WithRequestID(
	ctx context.Context,
	requestID string,
) context.Context {
	return context.WithValue(
		ctx,
		requestIDKey{},
		requestID,
	)
}
