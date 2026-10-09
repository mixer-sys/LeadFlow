package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLeadHandler_Create_InvalidJSON(t *testing.T) {
	h := NewLeadHandler(nil)

	body := strings.NewReader(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads", body)
	req.Header.Set("Content-Type", "applicaton/json")

	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", status)
	}
}

func TestLeadHandler_Create_missingSource(t *testing.T) {
	h := NewLeadHandler(nil)

	body := strings.NewReader(`{"email":"test@example.com","organization_id":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", status)
	}
}
