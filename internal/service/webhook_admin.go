package service

import (
	"context"
	"errors"
	"strings"

	"leadflow/internal/domain"
	"leadflow/internal/repository"
)

type WebhookAdminService struct {
	repo *repository.WebhookAdminRepo
}

func NewWebhookAdminService(
	repo *repository.WebhookAdminRepo,
) *WebhookAdminService {
	return &WebhookAdminService{repo: repo}
}

func (s *WebhookAdminService) List(
	ctx context.Context,
	organizationID int64,
) ([]domain.Webhook, error) {
	return s.repo.List(ctx, organizationID)
}

func (s *WebhookAdminService) Create(
	ctx context.Context,
	organizationID int64,
	url string,
	secret string,
	events string,
) (*domain.Webhook, error) {
	url = strings.TrimSpace(url)

	if url == "" {
		return nil, errors.New("webhook url is required")
	}

	if !strings.HasPrefix(url, "http://") &&
		!strings.HasPrefix(url, "https://") {
		return nil, errors.New("webhook url must start with http:// or htttps://")
	}

	if events == "" {
		events = "lead.created"
	}

	return s.repo.Create(
		ctx,
		repository.CreateWebhookParams{
			OrganizationID: organizationID,
			URL:            url,
			Secret:         secret,
			Events:         events,
			IsActive:       true,
		},
	)
}

func (s *WebhookAdminService) Toggle(
	ctx context.Context,
	id int64,
	organizationID int64,
) error {
	return s.repo.Toggle(ctx, id, organizationID)
}

func (s *WebhookAdminService) Delete(
	ctx context.Context,
	id int64,
	organizationID int64,
) error {
	return s.repo.Delete(ctx, id, organizationID)
}
