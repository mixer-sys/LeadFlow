package repository

import (
	"context"
	"time"

	"leadflow/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadRepo struct {
	pool *pgxpool.Pool
}

func NewLeadRepo(pool *pgxpool.Pool) *LeadRepo {
	return &LeadRepo{
		pool: pool,
	}
}

type CreateLeadParams struct {
	Source  string
	Name    *string
	Email   *string
	Phone   *string
	Message *string
}

func (r *LeadRepo) Create(ctx context.Context, params CreateLeadParams) (*domain.Lead, error) {
	query := `
	INSERT INTO leads (source, name, email, phone, message, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) RETURNING id, created_at, updated_at, source, name, email, phone, message, status, processed_at, telegram_message_id, telegram_sent_at
	`
	var lead domain.Lead

	err := r.pool.QueryRow(
		ctx,
		query,
		params.Source,
		params.Name,
		params.Email,
		params.Phone,
		params.Message,
		domain.LeadStatusNew,
	).Scan(
		&lead.ID,
		&lead.CreatedAt,
		&lead.UpdatedAt,
		&lead.Source,
		&lead.Name,
		&lead.Email,
		&lead.Phone,
		&lead.Message,
		&lead.Status,
		&lead.ProcessedAt,
		&lead.TelegramMessageID,
		&lead.TelegramSentAt,
	)
	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (r *LeadRepo) GetByID(ctx context.Context, id int64) (*domain.Lead, error) {
	query := `
		SELECT id, created_at, updated_at, source, name, email, phone, message, status, processed_at, telegram_message_id, telegram_sent_at, telegram_retry_count
		FROM leads
		WHERE id = $1
	`

	var lead domain.Lead

	row := r.pool.QueryRow(ctx, query, id)

	err := row.Scan(
		&lead.ID,
		&lead.CreatedAt,
		&lead.UpdatedAt,
		&lead.Source,
		&lead.Name,
		&lead.Email,
		&lead.Phone,
		&lead.Message,
		&lead.Status,
		&lead.ProcessedAt,
		&lead.TelegramMessageID,
		&lead.TelegramSentAt,
		&lead.TelegramRetryCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &lead, nil
}

func (r *LeadRepo) IncrTelegramRetryCount(
	ctx context.Context,
	id int64,
	maxRetries int,
) error {
	query := `
	UPDATE leads
	SET
		telegram_retry_count = telegram_retry_count + 1,
		status = CASE
			WHEN telegram_retry_count + 1 >= $2 THEN $3
		ELSE status
		END,
		updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(
		ctx,
		query,
		id,
		maxRetries,
		domain.LeadStatusFailed,
	)
	return err
}

func (r *LeadRepo) UpdateTelegramSent(
	ctx context.Context,
	id int64,
	messageID int64,
	sentAt time.Time,
) error {
	query := `
		UPDATE leads
		SET
			telegram_message_id = $2,
			telegram_sent_at = $3,
			status = $4,
			processed_at = $5,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		id,
		messageID,
		sentAt,
		domain.LeadStatusProcessed,
		sentAt,
	)
	return err
}
