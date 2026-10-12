package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"leadflow/internal/service"
)

type WebhookAdminHandler struct {
	svc *service.WebhookService
}

func NewWebhookAdminHandler(svc *service.WebhookService) *WebhookAdminHandler {
	return &WebhookAdminHandler{svc: svc}
}

func (h *WebhookAdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetDeliveryStats(r.Context())
	if err != nil {
		http.Error(
			w,
			`{"error":"internal error"}`,
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]int{
		"pending":       stats.Pending,
		"failed":        stats.Failed,
		"delivered":     stats.Delivered,
		"dead_letter":   stats.DeadLetter,
		"open_breakers": stats.OpenBreakers,
	})
}

func (h *WebhookAdminHandler) RetryDelivery(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(
			w,
			`{"error":"invalid delivery id"}`,
			http.StatusBadRequest,
		)
		return
	}

	if err := h.svc.RetryDeadLetterDelivery(r.Context(), deliveryID); err != nil {
		http.Error(
			w,
			`{"error":"delivery not found or not dead-letter"}`,
			http.StatusNotFound,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "retry_scheduled",
	})
}
