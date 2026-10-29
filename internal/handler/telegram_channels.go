package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"leadflow/internal/platform/telegram"
	"leadflow/internal/service"
)

type TelegramChannelsHandler struct {
	svc  *service.TelegramChannelService
	tmpl *template.Template
}

func NewTelegramChannelsHandler(
	svc *service.TelegramChannelService,
) (*TelegramChannelsHandler, error) {
	tmpl, err := template.ParseFiles(
		"templates/admin/telegram_channels.html",
	)
	if err != nil {
		return nil, err
	}

	return &TelegramChannelsHandler{
		svc:  svc,
		tmpl: tmpl,
	}, nil
}

type telegramChannelsPageData struct {
	Items []telegram.Channel
}

func (h *TelegramChannelsHandler) Page(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, _ := GetOrganizationID(r)

	channels, err := h.svc.List(r.Context(), organizationID)
	if err != nil {
		http.Error(w, "failed to load telegram channels", http.StatusInternalServerError)
		return
	}

	h.tmpl.ExecuteTemplate(w, "telegram_channels.html", telegramChannelsPageData{
		Items: channels,
	})
}

func (h *TelegramChannelsHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, _ := GetOrganizationID(r)

	_, err := h.svc.Create(r.Context(), telegram.Channel{
		OrganizationID: organizationID,
		Name:           r.FormValue("name"),
		BotToken:       r.FormValue("bot_token"),
		ChatID:         r.FormValue("chat_id"),
	})
	if err != nil {
		http.Error(w, "failed to create telegram channel", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/settings/telegram", http.StatusSeeOther)
}

func (h *TelegramChannelsHandler) Toggle(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, _ := GetOrganizationID(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err := h.svc.Toggle(r.Context(), organizationID, id); err != nil {
		http.Error(w, "failed to toggle telegram channel", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/settings/telegram", http.StatusSeeOther)
}

func (h *TelegramChannelsHandler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, _ := GetOrganizationID(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err := h.svc.Delete(r.Context(), organizationID, id); err != nil {
		http.Error(w, "failed to delete telegram channel", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/settings/telegram", http.StatusSeeOther)
}

func (h *TelegramChannelsHandler) SendTest(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, _ := GetOrganizationID(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	channel, err := h.svc.Get(r.Context(), organizationID, id)
	if err != nil {
		http.Error(w, "telegram channel not found", http.StatusNotFound)
		return
	}

	_, err = h.svc.SendMessage(
		r.Context(),
		channel.BotToken,
		channel.ChatID,
		"LeadFlow: тестовое сообщение из канала "+channel.Name,
	)
	if err != nil {
		http.Error(w, "failed to send telegram message", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/settings/telegram", http.StatusSeeOther)
}
