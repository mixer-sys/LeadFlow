package handler

import (
	"encoding/json"
	"fmt"
	"leadflow/internal/service"
	"net/http"
	"os"
)

type OrganizationHandler struct {
	svc *service.OrganizationService
}

func NewOrganizationHandler(svc *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{svc: svc}
}

type createOrganizationRequest struct {
	Name             string `json:"name"`
	TelegramBotToken string `json:"telegram_bot_token"`
	TelegramChatID   int64  `json:"telegram_chat_id"`
}

type createOrganizationResponse struct {
	ID int64 `json:"id"`
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	input := service.CreateOrganizationInput{
		Name:             req.Name,
		TelegramBotToken: req.TelegramBotToken,
		TelegramChatID:   req.TelegramChatID,
	}

	org, err := h.svc.CreateOrganization(r.Context(), input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create organization error: %v\n", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createOrganizationResponse{
		ID: org.ID,
	})

}
