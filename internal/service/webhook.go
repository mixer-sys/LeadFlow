package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"leadflow/internal/domain"
	"leadflow/internal/metrics"
	"leadflow/internal/repository"
)

const (
	maxWebhookAttempts      = 5
	webhookFailureThreshold = 5
	webhookBaseOpenDuration = 30 * time.Second
	webhookMaxOpenDuration  = 10 * time.Minute
)

func webhookOpenDuration(failureCount int) time.Duration {
	if failureCount < webhookFailureThreshold {
		return 0
	}

	attempt := failureCount - webhookFailureThreshold + 1

	const maxShift = 6

	if attempt > maxShift {
		attempt = maxShift
	}

	delay := webhookBaseOpenDuration * time.Duration(1<<(attempt-1))

	if delay > webhookMaxOpenDuration {
		delay = webhookMaxOpenDuration
	}

	return delay
}

func webhookRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	base := 5 * time.Second
	maxDelay := 5 * time.Minute

	delay := base * time.Duration(1<<(attempt-1))

	if delay > maxDelay {
		delay = maxDelay
	}

	jitter := time.Duration(rand.Int63n(int64(delay / 2)))

	return delay/2 + jitter
}

type WebhookService struct {
	repo *repository.WebhookRepo
}

func NewWebhookService(repo *repository.WebhookRepo) *WebhookService {
	return &WebhookService{repo: repo}
}

func (s *WebhookService) ListActiveWebhooks(
	ctx context.Context,
) ([]domain.Webhook, error) {
	return s.repo.ListActive(ctx)
}

func (s *WebhookService) SendWebhooks(
	ctx context.Context,
	lead *domain.Lead,
	requestID string,
) error {
	webhooks, err := s.ListActiveWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("list active webhooks: %w", err)
	}

	eventID := fmt.Sprintf("lead-%d-created", lead.ID)

	for _, webhook := range webhooks {
		if webhook.OpenedUntil != nil && webhook.OpenedUntil.After(time.Now()) {
			slog.InfoContext(
				ctx,
				"webhook circuit breaker is open, skipping",
				slog.String("request_id", requestID),
				slog.Int64("webhook_id", webhook.ID),
				slog.Int64("lead_id", lead.ID),
				slog.String("event_id", eventID),
				slog.Time("opened_until", *webhook.OpenedUntil),
			)

			continue
		}

		if err := s.SendWebhook(
			ctx,
			webhook,
			lead,
			eventID,
			requestID,
		); err != nil {
			slog.ErrorContext(
				ctx,
				"webhook delivery failed",
				slog.String("request_id", requestID),
				slog.String("error", err.Error()),
				slog.Int64("webhook_id", webhook.ID),
				slog.Int64("lead_id", lead.ID),
				slog.String("event_id", eventID),
			)

			continue
		}
	}

	return nil
}

func (s *WebhookService) GetDueDeliveries(
	ctx context.Context,
	limit int,
) ([]domain.WebhookDelivery, error) {
	return s.repo.FindDueDeliveries(ctx, limit)
}

func (s *WebhookService) SendWebhook(
	ctx context.Context,
	webhook domain.Webhook,
	lead *domain.Lead,
	eventID string,
	requestID string,
) error {
	delivery, err := s.repo.GetDelivery(
		ctx,
		webhook.ID,
		lead.ID,
		eventID,
	)
	if err != nil {
		return fmt.Errorf("get webhook delivery: %w", err)
	}

	if delivery != nil && delivery.Status == domain.WebhookDeliveryDelivered {
		slog.InfoContext(
			ctx,
			"webhook already delivered, skipping",
			slog.String("request_id", requestID),
			slog.Int64("webhook_id", webhook.ID),
			slog.Int64("lead_id", lead.ID),
			slog.String("event_id", eventID),
		)

		return nil
	}

	_, err = s.repo.CreateDelivery(
		ctx,
		webhook.ID,
		lead.ID,
		eventID,
		requestID,
		maxWebhookAttempts,
	)
	if err != nil {
		return fmt.Errorf("create webhook delivery: %w", err)
	}

	delivery, err = s.repo.GetDelivery(
		ctx,
		webhook.ID,
		lead.ID,
		eventID,
	)
	if err != nil {
		return fmt.Errorf("get webhook delivery after create: %w", err)
	}

	if delivery == nil {
		return fmt.Errorf("webhook delivery not found after create")
	}

	payload := map[string]interface{}{
		"event":    "lead.created",
		"event_id": eventID,
		"lead_id":  lead.ID,
		"source":   lead.Source,
		"name":     derefString(lead.Name),
		"email":    derefString(lead.Email),
		"phone":    derefString(lead.Phone),
		"message":  derefString(lead.Message),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		webhook.URL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}

	start := time.Now()

	resp, err := client.Do(req)

	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDelivery("error", duration)

		if handleErr := s.handleWebhookFailure(
			ctx,
			webhook,
			delivery,
			err.Error(),
		); handleErr != nil {
			return fmt.Errorf("handle webhook failure: %w", handleErr)
		}

		return fmt.Errorf("send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		metrics.ObserveDelivery("error", duration)

		err := fmt.Errorf("unexpected status: %s", resp.Status)

		if handleErr := s.handleWebhookFailure(
			ctx,
			webhook,
			delivery,
			err.Error(),
		); handleErr != nil {
			return fmt.Errorf("handle webhook failure: %w", handleErr)
		}

		return err
	}

	metrics.ObserveDelivery("success", duration)

	if err := s.repo.MarkDeliveryDelivered(ctx, delivery.ID); err != nil {
		return fmt.Errorf("mark delivery delivered: %w", err)
	}

	if err := s.repo.MarkWebhookSuccess(ctx, webhook.ID); err != nil {
		return fmt.Errorf("mark webhook success: %w", err)
	}

	metrics.WebhookCircuitBreakerOpen.Set(0)

	slog.InfoContext(
		ctx,
		"webhook delivered",
		slog.String("request_id", requestID),
		slog.Int64("delivery_id", delivery.ID),
		slog.Int64("webhook_id", webhook.ID),
		slog.Int64("lead_id", lead.ID),
		slog.String("event_id", eventID),
	)

	return nil
}

func (s *WebhookService) markDeliveryFailed(
	ctx context.Context,
	delivery *domain.WebhookDelivery,
	lastError string,
) error {
	metrics.WebhookDeliveryRetriesTotal.Inc()

	nextRetryAt := time.Now().Add(
		webhookRetryDelay(delivery.Attempts),
	)

	if delivery.Attempts+1 >= delivery.MaxAttempts {
		metrics.WebhookDeadLettersTotal.Inc()
	}

	err := s.repo.MarkDeliveryFailedWithRetry(
		ctx,
		delivery.ID,
		lastError,
		nextRetryAt,
		delivery.MaxAttempts,
	)

	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to mark webhook delivery failed",
			slog.String("request_id", delivery.RequestID),
			slog.Int64("delivery_id", delivery.ID),
			slog.Int64("lead_id", delivery.LeadID),
			slog.String("event_id", delivery.EventID),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}

func (s *WebhookService) handleWebhookFailure(
	ctx context.Context,
	webhook domain.Webhook,
	delivery *domain.WebhookDelivery,
	lastError string,
) error {
	if err := s.markDeliveryFailed(
		ctx,
		delivery,
		lastError,
	); err != nil {
		return err
	}

	failureCount := webhook.FailureCount + 1

	var openedUntil *time.Time

	if failureCount >= webhookFailureThreshold {
		openAt := time.Now().Add(
			webhookOpenDuration(failureCount),
		)

		openedUntil = &openAt

		metrics.WebhookCircuitBreakerOpen.Set(1)

		slog.WarnContext(
			ctx,
			"webhook circuit breaker opened",
			slog.String("request_id", delivery.RequestID),
			slog.Int64("webhook_id", webhook.ID),
			slog.Int("failure_count", failureCount),
			slog.Time("opened_until", openAt),
		)
	}

	return s.repo.MarkWebhookFailure(
		ctx,
		webhook.ID,
		failureCount,
		openedUntil,
	)
}

func (s *WebhookService) CreateWebhook(
	ctx context.Context,
	url string,
) (int64, error) {
	return s.repo.Create(ctx, url)
}

func (s *WebhookService) DeleteWebhook(
	ctx context.Context,
	id int64,
) error {
	return s.repo.Delete(ctx, id)
}

func (s *WebhookService) ToggleWebhookActive(
	ctx context.Context,
	id int64,
	isActive bool,
) error {
	return s.repo.ToggleActive(ctx, id, isActive)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func (s *WebhookService) ListActiveURLs(
	ctx context.Context,
) ([]string, error) {
	webhooks, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(webhooks))

	for _, webhook := range webhooks {
		urls = append(urls, webhook.URL)
	}

	return urls, nil
}

func (s *WebhookService) GetWebhookByID(
	ctx context.Context,
	id int64,
) (*domain.Webhook, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WebhookService) RetryDelivery(
	ctx context.Context,
	webhook *domain.Webhook,
	lead *domain.Lead,
	delivery *domain.WebhookDelivery,
) error {
	if webhook == nil {
		return fmt.Errorf("webhook %d not found", delivery.WebhookID)
	}

	if lead == nil {
		return fmt.Errorf("lead %d not found", delivery.LeadID)
	}

	if delivery.Status == domain.WebhookDeliveryDelivered {
		slog.InfoContext(
			ctx,
			"webhook delivery already delivered, skipping",
			slog.String("request_id", delivery.RequestID),
			slog.Int64("delivery_id", delivery.ID),
			slog.Int64("webhook_id", webhook.ID),
			slog.Int64("lead_id", lead.ID),
			slog.String("event_id", delivery.EventID),
		)

		return nil
	}

	if webhook.OpenedUntil != nil && webhook.OpenedUntil.After(time.Now()) {
		slog.InfoContext(
			ctx,
			"webhook circuit breaker is open, skipping retry",
			slog.String("request_id", delivery.RequestID),
			slog.Int64("delivery_id", delivery.ID),
			slog.Int64("webhook_id", webhook.ID),
			slog.Int64("lead_id", lead.ID),
			slog.String("event_id", delivery.EventID),
			slog.Time("opened_until", *webhook.OpenedUntil),
		)

		return nil
	}

	payload := map[string]interface{}{
		"event":    "lead.created",
		"event_id": delivery.EventID,
		"lead_id":  lead.ID,
		"source":   lead.Source,
		"name":     derefString(lead.Name),
		"email":    derefString(lead.Email),
		"phone":    derefString(lead.Phone),
		"message":  derefString(lead.Message),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		webhook.URL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}

	start := time.Now()

	resp, err := client.Do(req)

	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDelivery("error", duration)

		if handleErr := s.handleWebhookFailure(
			ctx,
			*webhook,
			delivery,
			err.Error(),
		); handleErr != nil {
			return fmt.Errorf("handle webhook failure: %w", handleErr)
		}

		return fmt.Errorf("send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		metrics.ObserveDelivery("error", duration)

		err := fmt.Errorf("unexpected status: %s", resp.Status)

		if handleErr := s.handleWebhookFailure(
			ctx,
			*webhook,
			delivery,
			err.Error(),
		); handleErr != nil {
			return fmt.Errorf("handle webhook failure: %w", handleErr)
		}

		return err
	}

	metrics.ObserveDelivery("success", duration)

	if err := s.repo.MarkDeliveryDelivered(ctx, delivery.ID); err != nil {
		return fmt.Errorf("mark delivery delivered: %w", err)
	}

	if err := s.repo.MarkWebhookSuccess(ctx, webhook.ID); err != nil {
		return fmt.Errorf("mark webhook success: %w", err)
	}

	metrics.WebhookCircuitBreakerOpen.Set(0)

	slog.InfoContext(
		ctx,
		"webhook delivery retried successfully",
		slog.String("request_id", delivery.RequestID),
		slog.Int64("delivery_id", delivery.ID),
		slog.Int64("webhook_id", webhook.ID),
		slog.Int64("lead_id", lead.ID),
		slog.String("event_id", delivery.EventID),
	)

	return nil
}

func (s *WebhookService) GetDeliveryStats(
	ctx context.Context,
) (*repository.DeliveryStats, error) {
	return s.repo.GetDeliveryStats(ctx)
}

func (s *WebhookService) RetryDeadLetterDelivery(
	ctx context.Context,
	deliveryID int64,
) error {
	return s.repo.ResetDeadLetterDelivery(ctx, deliveryID)
}
