package handler

import (
	"encoding/json"
	"leadflow/internal/service"
	"net/http"
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
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		http.Error(w, `{"error":"source is required"}`, http.StatusBadRequest)
		return
	}

	input := service.CreateLeadInput{
		Source:  req.Source,
		Name:    req.Name,
		Email:   &req.Email,
		Phone:   req.Phone,
		Message: req.Message,
	}

	lead, err := h.svc.CreateLead(r.Context(), input)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(createLeadResponse{ID: lead.ID})
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusBadRequest)
		return
	}
}
