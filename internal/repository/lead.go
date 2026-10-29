package repository

import (
	"context"
	"fmt"
	"time"

	"leadflow/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ListLeadsParams struct {
	Limit  int
	Offset int
}

type ListLeadsResult struct {
	Items []*domain.Lead
	Total int64
}

type LeadRepo struct {
	pool *pgxpool.Pool
}

func NewLeadRepo(pool *pgxpool.Pool) *LeadRepo {
	return &LeadRepo{
		pool: pool,
	}
}

type CreateLeadParams struct {
	Source         string
	Name           *string
	Email          *string
	Phone          *string
	Message        *string
	OrganizationID int64
}

func (r *LeadRepo) GetByID(ctx context.Context, id int64) (*domain.Lead, error) {
	query := `
		SELECT
			id,
			created_at,
			updated_at,
			source,
			name,
			email,
			phone,
			message,
			status,
			processed_at,
			telegram_message_id,
			telegram_sent_at,
			telegram_retry_count,
			organization_id,
			bitrix_id
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
		&lead.OrganizationID,
		&lead.BitrixID,
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
	failedStatus domain.LeadStatus,
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
		failedStatus,
	)
	return err
}

func (r *LeadRepo) ListLeads(
	ctx context.Context,
	params ListLeadsParams,
) (*ListLeadsResult, error) {
	var total int64

	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM leads").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count leads: %w", err)
	}

	query := `
		SELECT
			id,
			created_at,
			updated_at,
			source,
			name,
			email,
			phone,
			message,
			status,
			processed_at,
			telegram_message_id,
			telegram_sent_at,
			telegram_retry_count,
			organization_id,
			bitrix_id
		FROM leads
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.Lead, 0)

	for rows.Next() {
		var lead domain.Lead

		err := rows.Scan(
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
			&lead.OrganizationID,
			&lead.BitrixID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}

		items = append(items, &lead)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &ListLeadsResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *LeadRepo) Create(
	ctx context.Context,
	params CreateLeadParams,
) (*domain.Lead, error) {
	query := `
		INSERT INTO leads (
			source,
			name,
			email,
			phone,
			message,
			status,
			organization_id,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING
			id,
			created_at,
			updated_at,
			source,
			name,
			email,
			phone,
			message,
			status,
			processed_at,
			telegram_message_id,
			telegram_sent_at,
			telegram_retry_count,
			organization_id,
			bitrix_id
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
		params.OrganizationID,
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
		&lead.TelegramRetryCount,
		&lead.OrganizationID,
		&lead.BitrixID,
	)
	if err != nil {
		return nil, err
	}

	return &lead, nil
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

func (r *LeadRepo) FindByContact(
	ctx context.Context,
	orgID int64,
	email, phone *string,
) (*domain.Lead, error) {
	if email == nil && phone == nil {
		return nil, nil
	}

	query := `
		SELECT
			id,
			created_at,
			updated_at,
			source,
			name,
			email,
			phone,
			message,
			status,
			processed_at,
			telegram_message_id,
			telegram_sent_at,
			telegram_retry_count,
			organization_id,
			bitrix_id
		FROM leads
		WHERE organization_id = $1
			AND (
				($2::text IS NOT NULL AND email = $2::text)
				OR
				($3::text IS NOT NULL AND phone = $3::text)
			)
		ORDER BY id DESC
		LIMIT 1
	`

	var lead domain.Lead

	err := r.pool.QueryRow(ctx, query, orgID, email, phone).Scan(
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
		&lead.OrganizationID,
		&lead.BitrixID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &lead, nil
}

type UpdateLeadParams struct {
	ID             int64
	Name           *string
	Email          *string
	Phone          *string
	Message        *string
	Source         *string
	OrganizationID int64
}

func (r *LeadRepo) Update(
	ctx context.Context,
	params UpdateLeadParams,
) (*domain.Lead, error) {
	query := `
		UPDATE leads
		SET
			name = COALESCE($2, name),
			email = COALESCE($3, email),
			phone = COALESCE($4, phone),
			message = COALESCE($5, message),
			source = COALESCE($6, source),
			updated_at = NOW()
		WHERE id = $1 AND organization_id = $7
		RETURNING
			id,
			created_at,
			updated_at,
			source,
			name,
			email,
			phone,
			message,
			status,
			processed_at,
			telegram_message_id,
			telegram_sent_at,
			telegram_retry_count,
			organization_id,
			bitrix_id
	`

	var lead domain.Lead

	err := r.pool.QueryRow(
		ctx,
		query,
		params.ID,
		params.Name,
		params.Email,
		params.Phone,
		params.Message,
		params.Source,
		params.OrganizationID,
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
		&lead.TelegramRetryCount,
		&lead.OrganizationID,
		&lead.BitrixID,
	)

	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (r *LeadRepo) SetBitrixID(
	ctx context.Context,
	leadID int64,
	bitrixID int64,
) error {
	query := `
		UPDATE leads
		SET
			bitrix_id = $2,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, leadID, bitrixID)
	if err != nil {
		return fmt.Errorf("set bitrix id: %w", err)
	}

	return nil
}

func (r *LeadRepo) List(
	ctx context.Context,
	limit int32,
	afterID int64,
) ([]*domain.Lead, error) {
	if limit <= 0 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, source, name, email, phone, message, status,
		       bitrix_id, telegram_message_id, telegram_retry_count,
		       created_at, updated_at
		FROM leads
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2
	`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	leads := make([]*domain.Lead, 0, limit)

	for rows.Next() {
		lead := &domain.Lead{}

		err := rows.Scan(
			&lead.ID,
			&lead.Source,
			&lead.Name,
			&lead.Email,
			&lead.Phone,
			&lead.Message,
			&lead.Status,
			&lead.BitrixID,
			&lead.TelegramMessageID,
			&lead.TelegramRetryCount,
			&lead.CreatedAt,
			&lead.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}

		leads = append(leads, lead)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leads: %w", err)
	}

	return leads, nil
}

func (r *LeadRepo) Delete(
	ctx context.Context,
	id int64,
	organizationID int64,
) error {
	commandTag, err := r.pool.Exec(
		ctx,
		`
		DELETE FROM leads
		WHERE id = $1 AND organization_id = $2
		`,
		id,
		organizationID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrLeadNotFound
	}

	return nil
}

func (r *LeadRepo) UpdateStatus(
	ctx context.Context,
	id int64,
	organizationID int64,
	status domain.LeadStatus,
) (*domain.Lead, error) {
	rows, err := r.pool.Query(
		ctx,
		`UPDATE leads
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND organization_id = $2
		RETURNING
			id,
            created_at,
            updated_at,
            source,
            name,
            email,
            phone,
            message,
            status,
            processed_at,
            telegram_message_id,
            telegram_sent_at,
            telegram_retry_count,
            organization_id,
            bitrix_id
		`,
		id,
		organizationID,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}

		return nil, domain.ErrLeadNotFound
	}

	var lead domain.Lead

	err = rows.Scan(
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
		&lead.OrganizationID,
		&lead.BitrixID,
	)
	if err != nil {
		return nil, fmt.Errorf("scan lead: %w", err)
	}

	return &lead, nil
}

type ListForOrganizationParams struct {
	OrganizationID int64
	Status         *domain.LeadStatus
	Source         *string
	Search         string
	Sort           string
	Limit          int
	Offset         int
}

func (r *LeadRepo) ListForOrganization(
	ctx context.Context,
	organizationID int64,
	status *domain.LeadStatus,
	source *string,
	search string,
	sort string,
	limit int,
	offset int,
) ([]domain.Lead, int, error) {
	if limit <= 0 {
		limit = 50
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	orderBy := "created_at DESC"

	if sort == "created_at_asc" {
		orderBy = "created_at ASC"
	}

	where := "WHERE organization_id = $1"
	args := []any{organizationID}

	if status != nil {
		args = append(args, *status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}

	if source != nil && *source != "" {
		args = append(args, *source)
		where += fmt.Sprintf(" AND source = $%d", len(args))
	}

	if search != "" {
		args = append(args, "%"+search+"%")
		searchArg := len(args)

		where += fmt.Sprintf(
			` AND (
				COALESCE(name, '') ILIKE $%d
				OR COALESCE(email, '') ILIKE $%d
				OR COALESCE(phone, '') ILIKE $%d
			)`,
			searchArg,
			searchArg,
			searchArg,
		)
	}

	var total int

	err := r.pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM leads "+where,
		args...,
	).Scan(&total)

	if err != nil {
		return nil, 0, fmt.Errorf("count organization leads: %w", err)
	}

	query := `
		SELECT
			id,
			created_at,
			updated_at,
			source,
			name,
			email,
			phone,
			message,
			status,
			processed_at,
			telegram_message_id,
			telegram_sent_at,
			telegram_retry_count,
			organization_id,
			bitrix_id
		FROM leads
	` + where + fmt.Sprintf(
		" ORDER BY %s LIMIT $%d OFFSET $%d",
		orderBy,
		len(args)+1,
		len(args)+2,
	)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list organization leads: %w", err)
	}
	defer rows.Close()

	leads := make([]domain.Lead, 0, limit)

	for rows.Next() {
		var lead domain.Lead

		err := rows.Scan(
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
			&lead.OrganizationID,
			&lead.BitrixID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan lead: %w", err)
		}

		leads = append(leads, lead)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return leads, total, nil
}
