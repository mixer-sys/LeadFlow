package handler

import (
	"context"
	"net/http"

	"leadflow/internal/service"
)

type contextKey string

const (
	userIDKey         contextKey = "user_id"
	organizationIDKey contextKey = "organization_id"
)

func RequireSession(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := getUserIDFromSession(r)

			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
				return
			}

			user, err := auth.GetUserByID(r.Context(), userID)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, user.ID)
			ctx = context.WithValue(ctx, organizationIDKey, user.OrganizationID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) (int64, bool) {
	value, ok := r.Context().Value(userIDKey).(int64)

	return value, ok
}

func GetOrganizationID(r *http.Request) (int64, bool) {
	value, ok := r.Context().Value(organizationIDKey).(int64)

	return value, ok
}
