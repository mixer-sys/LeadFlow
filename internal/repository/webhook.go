package repository

import (
	"context"
	"fmt"
	"time"

	"leadflow/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxWebhookAttempts = 5

type WebhookRepo struct {
	pool *pgxpool.Pool
}

type DeliveryStats struct {
	Pending      int
	Failed       int
	Delivered    int
	DeadLetter   int
	OpenBreakers int
}

func (r *WebhookRepo) GetDeliveryStats(
	ctx context.Context,
) (*DeliveryStats, error) {
	query := `
		SELECT
			COUNT(d.id) FILTER (WHERE d.status = 'pending'),
			COUNT(d.id) FILTER (WHERE d.status = 'failed' AND d.next_retry_at IS NOT NULL),
			COUNT(d.id) FILTER (WHERE d.status = 'delivered'),
			COUNT(d.id) FILTER (WHERE d.status = 'failed' AND d.next_retry_at IS NULL),
			COUNT(w.id) FILTER (WHERE w.opened_until IS NOT NULL AND w.opened_until > NOW())
		FROM webhook_deliveries d
		LEFT JOIN webhooks w ON w.id = d.webhook_id
	`

	var stats DeliveryStats

	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.Pending,
		&stats.Failed,
		&stats.Delivered,
		&stats.DeadLetter,
		&stats.OpenBreakers,
	)

	if err != nil {
		return nil, fmt.Errorf("get delivery stats: %w", err)
	}

	return &stats, nil
}

func (r *WebhookRepo) ResetDeadLetterDelivery(
	ctx context.Context,
	deliveryID int64,
) error {
	query := `
		UPDATE webhook_deliveries
		SET
			status = $2,
			attempts = 0,
			next_retry_at = NOW(),
			last_error = NULL,
			updated_at = NOW()
		WHERE id = $1
			AND status = $3
			AND next_retry_at IS NULL
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		deliveryID,
		domain.WebhookDeliveryPending,
		domain.WebhookDeliveryFailed,
	)

	if err != nil {
		return fmt.Errorf("reset dead letter delivery: %w", err)
	}

	return nil
}

func NewWebhookRepo(pool *pgxpool.Pool) *WebhookRepo {
	return &WebhookRepo{pool: pool}
}

func (r *WebhookRepo) Create(
	ctx context.Context,
	url string,
) (int64, error) {
	query := `
		INSERT INTO webhooks (url, is_active, created_at, updated_at)
		VALUES ($1, true, NOW(), NOW())
		RETURNING id
	`

	var id int64
	err := r.pool.QueryRow(ctx, query, url).Scan(&id)
	return id, err
}

func (r *WebhookRepo) ListActive(
	ctx context.Context,
) ([]domain.Webhook, error) {
	query := `
		SELECT
			id,
			url,
			is_active,
			failure_count,
			opened_until,
			created_at,
			updated_at
		FROM webhooks
		WHERE is_active = true
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	webhooks := make([]domain.Webhook, 0)

	for rows.Next() {
		var webhook domain.Webhook

		err := rows.Scan(
			&webhook.ID,
			&webhook.URL,
			&webhook.IsActive,
			&webhook.FailureCount,
			&webhook.OpenedUntil,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, rows.Err()
}

func (r *WebhookRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *WebhookRepo) ToggleActive(
	ctx context.Context,
	id int64,
	isActive bool,
) error {
	query := `
		UPDATE webhooks
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, isActive)
	return err
}

func (r *WebhookRepo) GetDelivery(
	ctx context.Context,
	webhookID, leadID int64,
	eventID string,
) (*domain.WebhookDelivery, error) {
	query := `
		SELECT
			id,
			webhook_id,
			lead_id,
			event_id,
			status,
			attempts,
			last_error,
			created_at,
			updated_at
		FROM webhook_deliveries
		WHERE webhook_id = $1
			AND lead_id = $2
			AND event_id = $3
	`

	var delivery domain.WebhookDelivery

	err := r.pool.QueryRow(
		ctx,
		query,
		webhookID,
		leadID,
		eventID,
	).Scan(
		&delivery.ID,
		&delivery.WebhookID,
		&delivery.LeadID,
		&delivery.EventID,
		&delivery.Status,
		&delivery.Attempts,
		&delivery.LastError,
		&delivery.CreatedAt,
		&delivery.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &delivery, nil
}

func (r *WebhookRepo) CreateDelivery(
	ctx context.Context,
	webhookID, leadID int64,
	eventID string,
) (int64, error) {
	query := `
		INSERT INTO webhook_deliveries (
			webhook_id,
			lead_id,
			event_id,
			status,
			attempts,
			max_attempts,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, 1, $5, NOW(), NOW())
		ON CONFLICT (webhook_id, lead_id, event_id)
		DO UPDATE SET
			attempts = CASE
				WHEN webhook_deliveries.status = 'delivered'
					THEN webhook_deliveries.attempts
				WHEN webhook_deliveries.attempts >= webhook_deliveries.max_attempts
					THEN webhook_deliveries.attempts
				ELSE webhook_deliveries.attempts + 1
			END,
			updated_at = NOW()
		RETURNING id
	`

	var id int64
	err := r.pool.QueryRow(
		ctx,
		query,
		webhookID,
		leadID,
		eventID,
		domain.WebhookDeliveryPending,
		maxWebhookAttempts,
	).Scan(&id)

	return id, err
}

func (r *WebhookRepo) MarkDeliveryDelivered(
	ctx context.Context,
	deliveryID int64,
) error {
	query := `
		UPDATE webhook_deliveries
		SET
			status = $2,
			last_error = NULL,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		deliveryID,
		domain.WebhookDeliveryDelivered,
	)
	return err
}

func (r *WebhookRepo) MarkDeliveryFailed(
	ctx context.Context,
	deliveryID int64,
	lastError string,
) error {
	query := `
		UPDATE webhook_deliveries
		SET
			status = $2,
			last_error = $3,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		deliveryID,
		domain.WebhookDeliveryFailed,
		lastError,
	)
	return err
}

func (r *WebhookRepo) MarkDeliveryFailedWithRetry(
	ctx context.Context,
	deliveryID int64,
	lastError string,
	nextRetryAt time.Time,
	maxAttempts int,
) error {
	query := `
		UPDATE webhook_deliveries
		SET
			status = CASE
				WHEN attempts >= $2 THEN $3
				ELSE status
			END,
			last_error = $4,
			next_retry_at = CASE
				WHEN attempts >= $2 THEN NULL
				ELSE $5::timestamptz
			END,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		deliveryID,
		maxAttempts,
		domain.WebhookDeliveryFailed,
		lastError,
		nextRetryAt,
	)

	if err != nil {
		return fmt.Errorf("mark delivery failed with retry: %w", err)
	}

	return nil
}

func (r *WebhookRepo) FindDueDeliveries(
	ctx context.Context,
	limit int,
) ([]domain.WebhookDelivery, error) {
	query := `
		SELECT
			id,
			webhook_id,
			lead_id,
			event_id,
			status,
			attempts,
			last_error,
			next_retry_at,
			max_attempts,
			created_at,
			updated_at
		FROM webhook_deliveries
		WHERE status IN ('pending', 'failed')
			AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY next_retry_at NULLS FIRST, id
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deliveries := make([]domain.WebhookDelivery, 0)

	for rows.Next() {
		var delivery domain.WebhookDelivery

		err := rows.Scan(
			&delivery.ID,
			&delivery.WebhookID,
			&delivery.LeadID,
			&delivery.EventID,
			&delivery.Status,
			&delivery.Attempts,
			&delivery.LastError,
			&delivery.NextRetryAt,
			&delivery.MaxAttempts,
			&delivery.CreatedAt,
			&delivery.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		deliveries = append(deliveries, delivery)
	}

	return deliveries, rows.Err()
}

func (r *WebhookRepo) GetByID(
	ctx context.Context,
	id int64,
) (*domain.Webhook, error) {
	query := `
		SELECT
			id,
			url,
			is_active,
			failure_count,
			opened_until,
			created_at,
			updated_at
		FROM webhooks
		WHERE id = $1
	`

	var webhook domain.Webhook

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&webhook.ID,
		&webhook.URL,
		&webhook.IsActive,
		&webhook.FailureCount,
		&webhook.OpenedUntil,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &webhook, nil
}

func (r *WebhookRepo) MarkWebhookSuccess(
	ctx context.Context,
	webhookID int64,
) error {
	query := `
		UPDATE webhooks
		SET
			failure_count = 0,
			opened_until = NULL,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, webhookID)
	return err
}

func (r *WebhookRepo) MarkWebhookFailure(
	ctx context.Context,
	webhookID int64,
	failureCount int,
	openedUntil *time.Time,
) error {
	query := `
		UPDATE webhooks
		SET
			failure_count = $2,
			opened_until = $3,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		webhookID,
		failureCount,
		openedUntil,
	)
	return err
}
