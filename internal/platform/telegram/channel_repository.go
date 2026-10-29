package telegram

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type channelRepository struct {
	pool *pgxpool.Pool
}

func NewChannelRepository(pool *pgxpool.Pool) *channelRepository {
	return &channelRepository{pool: pool}
}

func (r *channelRepository) List(
	ctx context.Context,
	organizationID int64,
) ([]Channel, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT id, organization_id, name, bot_token, chat_id, is_active
		FROM telegram_channels
		WHERE organization_id = $1
		ORDER BY id DESC
		`,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []Channel

	for rows.Next() {
		var c Channel

		if err := rows.Scan(
			&c.ID,
			&c.OrganizationID,
			&c.Name,
			&c.BotToken,
			&c.ChatID,
			&c.IsActive,
		); err != nil {
			return nil, err
		}

		channels = append(channels, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

func (r *channelRepository) ListActive(
	ctx context.Context,
	organizationID int64,
) ([]Channel, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT id, organization_id, name, bot_token, chat_id, is_active
		FROM telegram_channels
		WHERE organization_id = $1 AND is_active = TRUE
		ORDER BY id
		`,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []Channel

	for rows.Next() {
		var c Channel

		if err := rows.Scan(
			&c.ID,
			&c.OrganizationID,
			&c.Name,
			&c.BotToken,
			&c.ChatID,
			&c.IsActive,
		); err != nil {
			return nil, err
		}

		channels = append(channels, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

func (r *channelRepository) Create(
	ctx context.Context,
	c Channel,
) (Channel, error) {
	err := r.pool.QueryRow(
		ctx,
		`
		INSERT INTO telegram_channels
			(organization_id, name, bot_token, chat_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_active
		`,
		c.OrganizationID,
		c.Name,
		c.BotToken,
		c.ChatID,
	).Scan(&c.ID, &c.IsActive)

	if err != nil {
		return Channel{}, err
	}

	return c, nil
}

func (r *channelRepository) Get(
	ctx context.Context,
	organizationID, id int64,
) (Channel, error) {
	var c Channel

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT id, organization_id, name, bot_token, chat_id, is_active
		FROM telegram_channels
		WHERE organization_id = $1 AND id = $2
		`,
		organizationID,
		id,
	).Scan(
		&c.ID,
		&c.OrganizationID,
		&c.Name,
		&c.BotToken,
		&c.ChatID,
		&c.IsActive,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Channel{}, errors.New("telegram channel not found")
	}

	if err != nil {
		return Channel{}, err
	}

	return c, nil
}

func (r *channelRepository) Toggle(
	ctx context.Context,
	organizationID, id int64,
) error {
	commandTag, err := r.pool.Exec(
		ctx,
		`
		UPDATE telegram_channels
		SET is_active = NOT is_active,
			updated_at = NOW()
		WHERE organization_id = $1 AND id = $2
		`,
		organizationID,
		id,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return errors.New("telegram channel not found")
	}

	return nil
}

func (r *channelRepository) Delete(
	ctx context.Context,
	organizationID, id int64,
) error {
	commandTag, err := r.pool.Exec(
		ctx,
		`
		DELETE FROM telegram_channels
		WHERE organization_id = $1 AND id = $2
		`,
		organizationID,
		id,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return errors.New("telegram channel not found")
	}

	return nil
}
