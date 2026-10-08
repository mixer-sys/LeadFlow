package middleware

import (
	"context"
	"net/http"

	"leadflow/internal/service"
)

type contextKey string

const apiKeyContextKey contextKey = "api_key"

func APIKeyAuth(svc *service.APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, `{"error":"missing X-API-Key header"}`, http.StatusUnauthorized)
				return
			}

			valid, err := svc.ValidateKey(r.Context(), apiKey)
			if err != nil || !valid {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), apiKeyContextKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAPIKeyFromContext(r *http.Request) string {
	if key, ok := r.Context().Value(apiKeyContextKey).(string); ok {
		return key
	}

	return ""
}
