package handler

import (
	"encoding/json"
	"fmt"
	"leadflow/internal/service"
	"net/http"
	"os"
	"strconv"
)

type LeadHandler struct {
	svc *service.LeadService
}

func NewLeadHandler(svc *service.LeadService) *LeadHandler {
	return &LeadHandler{
		svc: svc,
	}
}

type createLeadRequest struct {
	Source         string  `json:"source"`
	Name           *string `json:"name"`
	Email          string  `json:"email"`
	Phone          *string `json:"phone"`
	Message        *string `json:"message"`
	OrganizationID int64   `json:"organization_id"`
}

type LeadListHandler struct {
	svc *service.LeadService
}

type createLeadResponse struct {
	ID int64 `json:"id"`
}

func NewLeadListHandler(svc *service.LeadService) *LeadListHandler {
	return &LeadListHandler{svc: svc}
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
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
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

func (h *LeadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.Source == "" {

		http.Error(w, `{"error":"source is required"}`, http.StatusBadRequest)
		return
	}

	input := service.CreateLeadInput{
		Source:         req.Source,
		Name:           req.Name,
		Email:          &req.Email,
		Phone:          req.Phone,
		Message:        req.Message,
		OrganizationID: req.OrganizationID,
	}

	lead, err := h.svc.CreateLead(r.Context(), input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create lead error: %v\n", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(createLeadResponse{ID: lead.ID})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create lead error: %v\n", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusBadRequest)
		return
	}
}
