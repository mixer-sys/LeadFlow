package handler

import (
	"encoding/json"
	"net/http"

	"leadflow/internal/middleware"
	"leadflow/internal/service"
)

type APIKeyHandler struct {
	svc *service.APIKeyService
}

func NewAPIKeyHandler(svc *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{svc: svc}
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type CreateAPIKeyResponse struct {
	Key string `json:"key"`
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	organizationID, ok := middleware.GetOrganizationIDFromContext(r)
	if !ok {
		http.Error(
			w,
			`{"error":"organization not found"}`,
			http.StatusUnauthorized,
		)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(
			w,
			`{"error":"name is required"}`,
			http.StatusBadRequest,
		)
		return
	}

	key, err := h.svc.CreateAPIKey(
		r.Context(),
		req.Name,
		organizationID,
	)
	if err != nil {
		http.Error(
			w,
			`{"error":"internal error"}`,
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(CreateAPIKeyResponse{
		Key: key,
	})
}

type ListAPIKeysResponse struct {
	Keys []APIKeyDTO `json:"keys"`
}

type APIKeyDTO struct {
	ID             int64  `json:"id"`
	Key            string `json:"key"`
	Name           string `json:"name"`
	IsActive       bool   `json:"is_active"`
	OrganizationID int64  `json:"organization_id"`
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := h.svc.ListAPIKeys(r.Context())
	if err != nil {
		http.Error(
			w,
			`{"error":"internal error"}`,
			http.StatusInternalServerError,
		)
		return
	}

	dto := make([]APIKeyDTO, 0, len(keys))

	for _, k := range keys {
		dto = append(dto, APIKeyDTO{
			ID:             k.ID,
			Key:            k.Key,
			Name:           k.Name,
			IsActive:       k.IsActive,
			OrganizationID: k.OrganizationID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ListAPIKeysResponse{
		Keys: dto,
	})
}

func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.Error(w, "not implemented", http.StatusNotImplemented)
}
