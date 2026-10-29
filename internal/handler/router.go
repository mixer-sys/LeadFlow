package handler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RouterDeps struct {
	Lead         *LeadHandler
	LeadList     *LeadListHandler
	Organization *OrganizationHandler
	Webhook      *WebhookHandler
	APIKey       *APIKeyHandler
	WebhookAdmin *WebhookAdminHandler
	AdminLeads   *AdminLeadsHandler
	Health       *HealthHandler
	Auth         *AuthHandler

	APIKeyAuth     func(http.Handler) http.Handler
	RequireSession func(http.Handler) http.Handler

	Pages *PagesHandler

	TelegramChannels *TelegramChannelsHandler
	WebhookSettings  *WebhookSettingsHandler

	WebhookTest *WebhookTestHandler
}

func NewRouter(deps RouterDeps) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("GET /healthz", http.HandlerFunc(deps.Health.Live))
	mux.Handle("GET /readyz", deps.Health)

	mux.Handle(
		"POST /api/v1/leads",
		deps.APIKeyAuth(http.HandlerFunc(deps.Lead.Create)),
	)

	mux.Handle(
		"GET /api/v1/leads",
		deps.APIKeyAuth(deps.LeadList),
	)

	mux.Handle(
		"GET /api/v1/leads/{id}",
		deps.APIKeyAuth(http.HandlerFunc(deps.Lead.GetByID)),
	)

	mux.Handle(
		"PUT /api/v1/leads/{id}",
		deps.APIKeyAuth(http.HandlerFunc(deps.Lead.Put)),
	)

	mux.Handle(
		"PATCH /api/v1/leads/{id}",
		deps.APIKeyAuth(http.HandlerFunc(deps.Lead.UpdateStatus)),
	)

	mux.Handle(
		"DELETE /api/v1/leads/{id}",
		deps.APIKeyAuth(http.HandlerFunc(deps.Lead.Delete)),
	)

	mux.Handle(
		"GET /admin/leads",
		deps.RequireSession(deps.AdminLeads),
	)

	mux.Handle(
		"POST /admin/leads",
		deps.RequireSession(http.HandlerFunc(deps.AdminLeads.CreateLead)),
	)

	mux.Handle(
		"POST /admin/leads/{id}/status",
		deps.RequireSession(http.HandlerFunc(deps.AdminLeads.UpdateStatus)),
	)

	mux.Handle(
		"GET /admin/webhooks",
		deps.RequireSession(http.HandlerFunc(deps.WebhookSettings.Page)),
	)

	mux.Handle(
		"POST /admin/webhooks",
		deps.RequireSession(http.HandlerFunc(deps.WebhookSettings.Create)),
	)

	mux.Handle(
		"POST /admin/webhooks/{id}/toggle",
		deps.RequireSession(http.HandlerFunc(deps.WebhookSettings.Toggle)),
	)

	mux.Handle(
		"POST /admin/webhooks/{id}/delete",
		deps.RequireSession(http.HandlerFunc(deps.WebhookSettings.Delete)),
	)

	mux.Handle(
		"POST /admin/webhooks/{id}/test",
		deps.RequireSession(http.HandlerFunc(deps.WebhookTest.SendTest)),
	)

	mux.Handle(
		"POST /api/v1/organizations",
		http.HandlerFunc(deps.Organization.Create),
	)

	mux.Handle(
		"POST /api/v1/webhooks",
		http.HandlerFunc(deps.Webhook.Create),
	)

	mux.Handle(
		"GET /api/v1/webhooks",
		http.HandlerFunc(deps.Webhook.List),
	)

	mux.Handle(
		"DELETE /api/v1/webhooks/",
		http.HandlerFunc(deps.Webhook.Delete),
	)

	mux.Handle(
		"POST /api/v1/api-keys",
		http.HandlerFunc(deps.APIKey.Create),
	)

	mux.Handle(
		"GET /api/v1/api-keys",
		http.HandlerFunc(deps.APIKey.List),
	)

	mux.Handle(
		"DELETE /api/v1/api-keys/",
		http.HandlerFunc(deps.APIKey.Delete),
	)

	mux.Handle(
		"GET /admin/webhooks/deliveries/stats",
		deps.RequireSession(http.HandlerFunc(deps.WebhookAdmin.Stats)),
	)

	mux.HandleFunc("POST /api/v1/auth/register", deps.Auth.Register)
	mux.HandleFunc("POST /api/v1/auth/login", deps.Auth.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", deps.Auth.Logout)

	mux.Handle(
		"GET /api/v1/auth/me",
		deps.RequireSession(http.HandlerFunc(deps.Auth.Me)),
	)

	mux.Handle(
		"GET /admin",
		deps.RequireSession(http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			organizationID, _ := GetOrganizationID(r)

			writeJSON(w, http.StatusOK, map[string]any{
				"organization_id": organizationID,
				"message":         "welcome to LeadFlow admin",
			})
		})),
	)

	mux.HandleFunc("GET /register", deps.Pages.RegisterPage)
	mux.HandleFunc("POST /register", deps.Auth.Register)

	mux.HandleFunc("GET /login", deps.Pages.LoginPage)
	mux.HandleFunc("POST /login", deps.Auth.Login)

	mux.HandleFunc("POST /logout", deps.Auth.Logout)

	mux.Handle(
		"GET /admin/settings/telegram",
		deps.RequireSession(http.HandlerFunc(deps.TelegramChannels.Page)),
	)

	mux.Handle(
		"POST /admin/settings/telegram",
		deps.RequireSession(http.HandlerFunc(deps.TelegramChannels.Create)),
	)

	mux.Handle(
		"POST /admin/settings/telegram/{id}/toggle",
		deps.RequireSession(http.HandlerFunc(deps.TelegramChannels.Toggle)),
	)

	mux.Handle(
		"POST /admin/settings/telegram/{id}/delete",
		deps.RequireSession(http.HandlerFunc(deps.TelegramChannels.Delete)),
	)

	mux.Handle(
		"POST /admin/settings/telegram/{id}/test",
		deps.RequireSession(http.HandlerFunc(deps.TelegramChannels.SendTest)),
	)

	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	return mux
}
