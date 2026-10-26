package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
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

	GetByID(
		ctx context.Context,
		id int64,
	) (*domain.Lead, error)

	UpdateLead(
		ctx context.Context,
		input service.UpdateLeadInput,
	) (*domain.Lead, error)

	DeleteLead(
		ctx context.Context,
		id int64,
		organizationID int64,
	) error
	UpdateStatus(
		ctx context.Context,
		id int64,
		organizationID int64,
		status domain.LeadStatus,
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

type updateStatusRequest struct {
	Status string `json:"status"`
}

type createLeadResponse struct {
	ID int64 `json:"id"`
}

func (h *LeadHandler) UpdateStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid lead id",
		})
		return
	}

	organizationID, ok := auth.GetOrganizationIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "organization not found",
		})
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Status == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "status is required",
		})
		return
	}

	status := domain.LeadStatus(req.Status)
	if !status.IsValid() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid status",
		})
		return
	}

	lead, err := h.svc.UpdateStatus(
		r.Context(),
		id,
		organizationID,
		status,
	)
	if err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "lead not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "update lead status failed",
		})
		return
	}
	writeJSON(w, http.StatusOK, lead)
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

	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "email is required",
		})
		return
	}

	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid email",
			})
			return
		}
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

func (h *LeadHandler) Put(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid lead id",
		})
		return
	}

	organizationID, ok := auth.GetOrganizationIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "organization not found",
		})
		return
	}

	var req struct {
		Name    *string `json:"name"`
		Email   *string `json:"email"`
		Phone   *string `json:"phone"`
		Message *string `json:"message"`
		Source  *string `json:"source"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON body",
		})
		return
	}

	lead, err := h.svc.UpdateLead(r.Context(), service.UpdateLeadInput{
		ID:             id,
		Name:           req.Name,
		Email:          req.Email,
		Phone:          req.Phone,
		Message:        req.Message,
		Source:         req.Source,
		OrganizationID: organizationID,
	})

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrLeadNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "lead not found",
			})
		case errors.Is(err, domain.ErrInvalidEmail):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid email",
			})

		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "update lead failed",
			})
		}

		return
	}

	writeJSON(w, http.StatusOK, lead)
}

func (h *LeadHandler) GetByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid lead id",
		})
		return
	}

	lead, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "lead not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "get lead failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, lead)
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

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(
			w,
			`{"error":"failed to encode response"}`,
			http.StatusInternalServerError,
		)
	}
}

func (h *LeadHandler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid lead id",
		})
		return
	}

	organizationID, ok := auth.GetOrganizationIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "organization not found",
		})
		return
	}

	if err := h.svc.DeleteLead(r.Context(), id, organizationID); err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "lead not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "delete lead failed",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
