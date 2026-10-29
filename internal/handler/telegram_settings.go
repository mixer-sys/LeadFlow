package handler

import (
	"html/template"
	"net/http"

	"leadflow/internal/service"
)

type TelegramSettingsHandler struct {
	svc *service.TelegramSettingsService
	tpl *template.Template
}

func NewTelegramSettingsHandler(
	svc *service.TelegramSettingsService,
) (*TelegramSettingsHandler, error) {
	tpl, err := template.ParseFiles(
		"templates/admin/telegram_settings.html",
	)
	if err != nil {
		return nil, err
	}

	return &TelegramSettingsHandler{
		svc: svc,
		tpl: tpl,
	}, nil
}

type telegramSettingsPageData struct {
	Enabled bool
	ChatID  string
}

func (h *TelegramSettingsHandler) Page(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	settings, err := h.svc.Get(r.Context(), organizationID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := telegramSettingsPageData{
		Enabled: settings.Enabled,
		ChatID:  settings.ChatID,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}

}

func (h *TelegramSettingsHandler) Save(
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

	err := h.svc.Save(r.Context(), service.SaveTelegramSettingsInput{
		OrganizationID: organizationID,
		Enabled:        r.FormValue("enabled") == "on",
		BotToken:       r.FormValue("bot_token"),
		ChatID:         r.FormValue("chat_id"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin/settings/telegram", http.StatusSeeOther)
}

func (h *TelegramSettingsHandler) SendTest(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.svc.SendTestMessage(r.Context(), organizationID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin/settings/telegram", http.StatusSeeOther)
}
