package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookRepo(pool *pgxpool.Pool) *WebhookRepo {
	return &WebhookRepo{pool: pool}
}

func (r *WebhookRepo) Create(ctx context.Context, url string) (int64, error) {
	query := `
		INSERT INTO webhooks (url, is_active, created_at, updated_at)
		VALUES ($1, true, NOW(), NOW())
		RETURNING id
	`
	var id int64
	err := r.pool.QueryRow(ctx, query, url).Scan(&id)
	return id, err
}

func (r *WebhookRepo) ListActive(ctx context.Context) ([]string, error) {
	query := `SELECT url FROM webhooks WHERE is_active = true ORDER BY id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	urls := make([]string, 0)
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	return urls, rows.Err()
}

func (r *WebhookRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *WebhookRepo) ToggleActive(ctx context.Context, id int64, isActive bool) error {
	query := `
		UPDATE webhooks
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, isActive)
	return err
}
