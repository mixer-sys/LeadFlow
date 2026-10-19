package grpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	log "leadflow/internal/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requestIDKey = "x-request-id"

func RequestIDUnaryServerInterceptor(
	logger *slog.Logger,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		requestID := ""

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(requestIDKey); len(values) > 0 {
				requestID = values[0]
			}
		}

		if requestID == "" {
			generated, err := generateRequestID()
			if err != nil {
				return nil, status.Error(
					codes.Internal,
					"failed to generate request id",
				)
			}

			requestID = generated
		}

		ctx = log.WithRequestID(ctx, requestID)

		resp, err := handler(ctx, req)

		logger.InfoContext(
			ctx,
			"grpc request completed",
			"request_id", requestID,
			"method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"grpc_code", status.Code(err).String(),
		)

		return resp, err
	}
}

func generateRequestID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(bytes[0:4]),
		hex.EncodeToString(bytes[4:6]),
		hex.EncodeToString(bytes[6:8]),
		hex.EncodeToString(bytes[8:10]),
		hex.EncodeToString(bytes[10:16]),
	), nil
}
