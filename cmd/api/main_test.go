package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"leadflow/internal/handler"
)

func TestHealthHandler(t *testing.T) {
	healthHandler := handler.NewHealthHandler(nil, nil)

	router := handler.NewRouter(handler.RouterDeps{
		Health: healthHandler,

		APIKeyAuth: func(next http.Handler) http.Handler {
			return next
		},

		RequireSession: func(next http.Handler) http.Handler {
			return next
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	expected := "{\"status\":\"ok\"}\n"

	if body := rr.Body.String(); body != expected {
		t.Errorf("expected body %q, got %q", expected, body)
	}
}
