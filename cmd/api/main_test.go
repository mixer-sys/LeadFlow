package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	healthHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	expected := `{"status":"ok"}`
	if body := rr.Body.String(); body != expected {
		t.Errorf("expected body %q, got %q", expected, body)
	}
}

func TestStatusWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rr}

	sw.WriteHeader(http.StatusCreated)

	if sw.status != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, sw.status)
	}
}

func TestStatusWriter_DefaultStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	sw := &statusWriter{
		ResponseWriter: rr,
	}

	_, _ = sw.Write([]byte("ok"))

	if sw.status != http.StatusOK {
		t.Errorf("expected default status %d, got %d", http.StatusOK, sw.status)
	}
}
