package handler

import (
	"context"
	"encoding/json"
	"leadflow/internal/domain"
	"leadflow/internal/repository"
	"leadflow/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

type mockLeadLister struct {
	result *repository.ListLeadsResult
	err    error
}

func (m *mockLeadLister) ListLeads(ctx context.Context, input service.ListLeadsInput) (*repository.ListLeadsResult, error) {
	return m.result, m.err
}

func TestLeadListHandler_Success(t *testing.T) {

	now := time.Now()

	mock := &mockLeadLister{
		result: &repository.ListLeadsResult{
			Items: []*domain.Lead{
				{
					ID:        1,
					Source:    "web",
					Status:    domain.LeadStatusNew,
					CreatedAt: now,
				},
			},
			Total: 1,
		},
	}

	h := NewLeadListHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	var resp listLeadsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Items))
	}

	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
}
