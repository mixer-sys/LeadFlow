package repository

import (
	"context"
	"leadflow/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyRepo struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{pool: pool}
}

func (r *APIKeyRepo) Create(ctx context.Context, key, name string) (int64, error) {
	query := `
		INSERT INTO api_keys (key, name, is_active, created_at, updated_at)
		VALUES ($1, $2, true, NOW(), NOW())
		RETURNING id
	`

	var id int64
	err := r.pool.QueryRow(ctx, query, key, name).Scan(&id)
	return id, err
}

func (r *APIKeyRepo) GetByKey(ctx context.Context, key string) (*string, error) {
	query := `
		SELECT name
		FROM api_keys
		WHERE key = $1 AND is_active = true
	`

	var name string
	err := r.pool.QueryRow(ctx, query, key).Scan(&name)
	if err != nil {
		return nil, err
	}

	return &name, nil
}

func (r *APIKeyRepo) List(ctx context.Context) ([]domain.APIKey, error) {
	query := `
		SELECT id, key, name, is_active, created_at, updated_at
		FROM api_keys
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]domain.APIKey, 0)
	for rows.Next() {
		var k domain.APIKey
		if err := rows.Scan(&k.ID, &k.Key, &k.Name, &k.IsActive, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}

	return keys, rows.Err()
}

func (r *APIKeyRepo) Delete(ctx context.Context, id int64) error {
	query := `
		DELETE FROM api_keys
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *APIKeyRepo) ToggleActive(ctx context.Context, id int64, isActive bool) error {
	query := `
		UPDATE api_keys
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, isActive)
	return err
}
