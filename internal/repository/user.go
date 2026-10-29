package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"leadflow/internal/domain"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) CreateWithOrganization(
	ctx context.Context,
	organizationName string,
	email string,
	passwordHash string,
) (*domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var organizationID int64

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO organizations (name)
		VALUES ($1)
		RETURNING id
		`,
		organizationName,
	).Scan(&organizationID)
	if err != nil {
		return nil, err
	}

	var user domain.User

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO users (
			organization_id,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			organization_id,
			email,
			password_hash,
			role,
			created_at,
			updated_at
		`,
		organizationID,
		email,
		passwordHash,
		domain.UserRoleOwner,
	).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) GetByEmail(
	ctx context.Context,
	email string,
) (*domain.User, error) {
	var user domain.User

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			organization_id,
			email,
			password_hash,
			role,
			created_at,
			updated_at
		FROM users
		WHERE email = $1
		`,
		email,
	).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) GetByID(
	ctx context.Context,
	id int64,
) (*domain.User, error) {
	var user domain.User

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT 
			id,
			organization_id,
			email,
			password_hash,
			role,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
		`,
		id,
	).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}
