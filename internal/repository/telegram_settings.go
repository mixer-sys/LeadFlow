package repository

import (
	"context"
	"errors"

	"leadflow/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelegramSettingsRepo struct {
	pool *pgxpool.Pool
}

func NewTelegramSettingsRepo(
	pool *pgxpool.Pool,
) *TelegramSettingsRepo {
	return &TelegramSettingsRepo{pool: pool}
}

func (r *TelegramSettingsRepo) Get(
	ctx context.Context,
	organizationID int64,
) (*domain.TelegramSettings, error) {
	var settings domain.TelegramSettings

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT
			organization_id,
			telegram_enabled,
			telegram_bot_token_encrypted,
			telegram_chat_id,
			created_at,
			updated_at
		FROM organization_settings
		WHERE organization_id = $1
		`,
		organizationID,
	).Scan(
		&settings.OrganizationID,
		&settings.Enabled,
		&settings.BotTokenEncrypted,
		&settings.ChatID,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.TelegramSettings{
				OrganizationID: organizationID,
			}, nil
		}

		return nil, err
	}

	return &settings, nil
}

func (r *TelegramSettingsRepo) Save(
	ctx context.Context,
	settings *domain.TelegramSettings,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO organization_settings (
			organization_id,
			telegram_enabled,
			telegram_bot_token_encrypted,
			telegram_chat_id
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (organization_id)
		DO UPDATE SET
			telegram_enabled = EXCLUDED.telegram_enabled,
			telegram_bot_token_encrypted = EXCLUDED.telegram_bot_token_encrypted,
			telegram_chat_id = EXCLUDED.telegram_chat_id,
			updated_at = NOW()
		`,
		settings.OrganizationID,
		settings.Enabled,
		settings.BotTokenEncrypted,
		settings.ChatID,
	)
	return err
}
