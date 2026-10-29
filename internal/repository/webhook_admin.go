package repository

import (
	"context"

	"leadflow/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookAdminRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookAdminRepo(
	pool *pgxpool.Pool,
) *WebhookAdminRepo {
	return &WebhookAdminRepo{pool: pool}
}

type CreateWebhookParams struct {
	OrganizationID int64
	URL            string
	Secret         string
	Events         string
	IsActive       bool
}

func (r *WebhookAdminRepo) List(
	ctx context.Context,
	organizationID int64,
) ([]domain.Webhook, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			organization_id,
			url,
			secret,
			events,
			is_active,
			failure_count,
			opened_until,
			created_at,
			updated_at
		FROM webhooks
		WHERE organization_id = $1
		ORDER BY id DESC
		`,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Webhook, 0)

	for rows.Next() {
		var webhook domain.Webhook

		err := rows.Scan(
			&webhook.ID,
			&webhook.OrganizationID,
			&webhook.URL,
			&webhook.Secret,
			&webhook.Events,
			&webhook.IsActive,
			&webhook.FailureCount,
			&webhook.OpenedUntil,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, webhook)
	}

	return items, rows.Err()
}

func (r *WebhookAdminRepo) Create(
	ctx context.Context,
	params CreateWebhookParams,
) (*domain.Webhook, error) {
	var webhook domain.Webhook

	err := r.pool.QueryRow(
		ctx,
		`
		INSERT INTO webhooks (
			organization_id,
			url,
			secret,
			events,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id,
			organization_id,
			url,
			secret,
			events,
			is_active,
			failure_count,
			opened_until,
			created_at,
			updated_at
		`,
		params.OrganizationID,
		params.URL,
		params.Secret,
		params.Events,
		params.IsActive,
	).Scan(
		&webhook.ID,
		&webhook.OrganizationID,
		&webhook.URL,
		&webhook.Secret,
		&webhook.Events,
		&webhook.IsActive,
		&webhook.FailureCount,
		&webhook.OpenedUntil,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (r *WebhookAdminRepo) Toggle(
	ctx context.Context,
	id int64,
	organizationID int64,
) error {
	commandTag, err := r.pool.Exec(
		ctx,
		`
		UPDATE webhooks
		SET is_active = NOT is_active, updated_at = NOW()
		WHERE id = $1 AND organization_id = $2
		`,
		id,
		organizationID,
	)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrWebhookNotFound
	}

	return nil
}

func (r *WebhookAdminRepo) Delete(
	ctx context.Context,
	id int64,
	organizationID int64,
) error {
	commandTag, err := r.pool.Exec(
		ctx,
		`
		DELETE FROM webhooks
		WHERE id = $1 AND organization_id = $2
		`,
		id,
		organizationID,
	)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrWebhookNotFound
	}

	return nil
}
