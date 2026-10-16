package grpc

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authorizationKey = "authorization"
const bearerPrefix = "Bearer "

func AuthUnaryServerInterceptor(
	logger *slog.Logger,
	internalToken string,
) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if internalToken == "" {
			logger.Error(
				"grpc internal token is not configured",
				"method", info.FullMethod,
			)

			return nil, status.Error(
				codes.Unauthenticated,
				"service authentication is not configured",
			)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(
				codes.Unauthenticated,
				"metadata is not provided",
			)
		}

		values := md.Get(authorizationKey)
		if len(values) == 0 {
			return nil, status.Error(
				codes.Unauthenticated,
				"authorization token is not provided",
			)
		}

		authHeader := values[0]

		if !strings.HasPrefix(authHeader, bearerPrefix) {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid authorization format",
			)
		}

		token := strings.TrimPrefix(authHeader, bearerPrefix)

		if subtle.ConstantTimeCompare(
			[]byte(token),
			[]byte(internalToken),
		) != 1 {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid authorization token",
			)
		}

		return handler(ctx, req)
	}
}
