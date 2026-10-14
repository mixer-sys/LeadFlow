package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"leadflow/internal/auth"
	"leadflow/internal/domain"
	"leadflow/internal/middleware"
	"leadflow/internal/repository"
	"leadflow/internal/service"
)

type LeadCreator interface {
	CreateLead(
		ctx context.Context,
		input service.CreateLeadInput,
		requestID string,
	) (*domain.Lead, error)
}

type LeadLister interface {
	ListLeads(
		ctx context.Context,
		input service.ListLeadsInput,
	) (*repository.ListLeadsResult, error)
}

type LeadHandler struct {
	svc LeadCreator
}

func NewLeadHandler(svc LeadCreator) *LeadHandler {
	return &LeadHandler{
		svc: svc,
	}
}

type createLeadRequest struct {
	Source  string  `json:"source"`
	Name    *string `json:"name"`
	Email   string  `json:"email"`
	Phone   *string `json:"phone"`
	Message *string `json:"message"`
}

type createLeadResponse struct {
	ID int64 `json:"id"`
}

func (h *LeadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLeadRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			`{"error":"invalid json"}`,
			http.StatusBadRequest,
		)

		return
	}

	if req.Source == "" {
		http.Error(
			w,
			`{"error":"source is required"}`,
			http.StatusBadRequest,
		)

		return
	}

	organizationID, ok := auth.GetOrganizationIDFromContext(r)
	if !ok {
		http.Error(
			w,
			`{"error":"organization not found"}`,
			http.StatusUnauthorized,
		)

		return
	}

	requestID := middleware.GetRequestID(r)

	input := service.CreateLeadInput{
		Source:         req.Source,
		Name:           req.Name,
		Email:          &req.Email,
		Phone:          req.Phone,
		Message:        req.Message,
		OrganizationID: organizationID,
	}

	lead, err := h.svc.CreateLead(
		r.Context(),
		input,
		requestID,
	)
	if err != nil {
		slog.ErrorContext(
			r.Context(),
			"failed to create lead",
			slog.String("request_id", requestID),
			slog.Int64("organization_id", organizationID),
			slog.String("source", req.Source),
			slog.String("error", err.Error()),
		)

		http.Error(
			w,
			`{"error":"internal error"}`,
			http.StatusInternalServerError,
		)

		return
	}

	slog.InfoContext(
		r.Context(),
		"lead received",
		slog.String("request_id", requestID),
		slog.Int64("lead_id", lead.ID),
		slog.String("source", lead.Source),
		slog.Bool("has_email", lead.Email != nil),
		slog.Bool("has_phone", lead.Phone != nil),
		slog.Bool("has_message", lead.Message != nil),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createLeadResponse{
		ID: lead.ID,
	}); err != nil {
		slog.ErrorContext(
			r.Context(),
			"failed to encode response",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
	}
}

type LeadListHandler struct {
	svc LeadLister
}

type leadListItem struct {
	ID        int64  `json:"id"`
	Source    string `json:"source"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type listLeadsResponse struct {
	Items []leadListItem `json:"items"`
	Total int64          `json:"total"`
}

func NewLeadListHandler(svc LeadLister) *LeadListHandler {
	return &LeadListHandler{svc: svc}
}

func (h *LeadListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	result, err := h.svc.ListLeads(r.Context(), service.ListLeadsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		slog.ErrorContext(
			r.Context(),
			"failed to list leads",
			slog.String("request_id", middleware.GetRequestID(r)),
			slog.String("error", err.Error()),
		)

		http.Error(
			w,
			`{"error":"internal error"}`,
			http.StatusInternalServerError,
		)

		return
	}

	items := make([]leadListItem, 0, len(result.Items))

	for _, lead := range result.Items {
		name := ""
		if lead.Name != nil {
			name = *lead.Name
		}

		email := ""
		if lead.Email != nil {
			email = *lead.Email
		}

		phone := ""
		if lead.Phone != nil {
			phone = *lead.Phone
		}

		items = append(items, leadListItem{
			ID:        lead.ID,
			Source:    lead.Source,
			Name:      name,
			Email:     email,
			Phone:     phone,
			Status:    string(lead.Status),
			CreatedAt: lead.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	resp := listLeadsResponse{
		Items: items,
		Total: result.Total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(resp)
}
