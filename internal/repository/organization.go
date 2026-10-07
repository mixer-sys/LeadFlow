package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"leadflow/internal/domain"
)

type OrganizationRepo struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepo(pool *pgxpool.Pool) *OrganizationRepo {
	return &OrganizationRepo{pool: pool}
}

type CreateOrganizationParams struct {
	Name             string
	TelegramBotToken string
	TelegramChatID   int64
}

func (r *OrganizationRepo) Create(ctx context.Context, params CreateOrganizationParams) (*domain.Organization, error) {
	query := `
		INSERT INTO organizations (name, telegram_bot_token, telegram_chat_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at, name, telegram_bot_token, telegram_chat_id
	`

	var org domain.Organization

	err := r.pool.QueryRow(
		ctx,
		query,
		params.Name,
		params.TelegramBotToken,
		params.TelegramChatID,
	).Scan(
		&org.ID,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.Name,
		&org.TelegramBotToken,
		&org.TelegramChatID,
	)
	if err != nil {
		return nil, err
	}

	return &org, nil
}

func (r *OrganizationRepo) GetByID(ctx context.Context, id int64) (*domain.Organization, error) {
	query := `
		SELECT id, created_at, updated_at, name, telegram_bot_token, telegram_chat_id
		FROM organizations
		WHERE ID = $1
	`

	var org domain.Organization

	row := r.pool.QueryRow(ctx, query, id)

	err := row.Scan(
		&org.ID,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.Name,
		&org.TelegramBotToken,
		&org.TelegramChatID,
	)
	if err != nil {
		return nil, err
	}

	return &org, nil
}
