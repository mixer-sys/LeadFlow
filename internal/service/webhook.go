package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"leadflow/internal/domain"
	"leadflow/internal/repository"
	"net/http"
	"time"
)

type WebhookService struct {
	repo *repository.WebhookRepo
}

func NewWebhookService(repo *repository.WebhookRepo) *WebhookService {
	return &WebhookService{repo: repo}
}

func (s *WebhookService) ListActiveURLs(ctx context.Context) ([]string, error) {
	return s.repo.ListActive(ctx)
}

func (s *WebhookService) SendWebhooks(ctx context.Context, lead *domain.Lead) error {
	urls, err := s.ListActiveURLs(ctx)
	if err != nil {
		return fmt.Errorf("list active webhooks: %w", err)
	}

	for _, url := range urls {
		if err := s.SendWebhook(ctx, url, lead); err != nil {
			// logging
			continue
		}
	}
	return nil
}

func (s *WebhookService) SendWebhook(ctx context.Context, url string, lead *domain.Lead) error {
	payload := map[string]interface{}{
		"event":   "lead.created",
		"lead_id": lead.ID,
		"source":  lead.Source,
		"name":    derefString(lead.Name),
		"email":   derefString(lead.Email),
		"phone":   derefString(lead.Phone),
		"message": derefString(lead.Message),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}

func (s *WebhookService) CreateWebhook(ctx context.Context, url string) (int64, error) {
	return s.repo.Create(ctx, url)
}

func (s *WebhookService) DeleteWebhook(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *WebhookService) ToggleWebhookActive(ctx context.Context, id int64, isActive bool) error {
	return s.repo.ToggleActive(ctx, id, isActive)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
