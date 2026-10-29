package handler

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"leadflow/internal/service"
)

type WebhookSettingsHandler struct {
	svc *service.WebhookAdminService
	tpl *template.Template
}

func NewWebhookSettingsHandler(
	svc *service.WebhookAdminService,
) (*WebhookSettingsHandler, error) {
	tpl, err := template.ParseFiles(
		"templates/admin/webhooks.html",
	)
	if err != nil {
		return nil, err
	}

	return &WebhookSettingsHandler{
		svc: svc,
		tpl: tpl,
	}, nil
}

type webhookItem struct {
	ID       int64
	URL      string
	Events   string
	IsActive bool
}

type webhooksPageData struct {
	Items []webhookItem
}

func (h *WebhookSettingsHandler) Page(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	webhooks, err := h.svc.List(r.Context(), organizationID)
	if err != nil {
		slog.ErrorContext(
			r.Context(),
			"failed to list webhooks",
			slog.String("error", err.Error()),
			slog.Int64("organization_id", organizationID),
		)

		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	items := make([]webhookItem, 0, len(webhooks))

	for _, webhook := range webhooks {
		items = append(items, webhookItem{
			ID:       webhook.ID,
			URL:      webhook.URL,
			Events:   webhook.Events,
			IsActive: webhook.IsActive,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tpl.Execute(w, webhooksPageData{
		Items: items,
	}); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebhookSettingsHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	_, err := h.svc.Create(
		r.Context(),
		organizationID,
		r.FormValue("url"),
		r.FormValue("secret"),
		r.FormValue("events"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin/webhooks", http.StatusSeeOther)
}

func (h *WebhookSettingsHandler) Toggle(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid webhook id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Toggle(r.Context(), id, organizationID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin/webhooks", http.StatusSeeOther)
}

func (h *WebhookSettingsHandler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid webhook id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(r.Context(), id, organizationID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin/webhooks", http.StatusSeeOther)
}
