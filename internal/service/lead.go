package service

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"

	"leadflow/internal/domain"
	"leadflow/internal/platform/queue"
	"leadflow/internal/repository"

	"github.com/jackc/pgx/v5"
)

type LeadService struct {
	repo  *repository.LeadRepo
	queue *queue.RedisQueue
}

func NewLeadService(
	repo *repository.LeadRepo,
	q *queue.RedisQueue,
) *LeadService {
	return &LeadService{
		repo:  repo,
		queue: q,
	}
}

type CreateLeadInput struct {
	Source         string
	Name           *string
	Email          *string
	Phone          *string
	Message        *string
	OrganizationID int64
}

type UpdateLeadInput struct {
	ID             int64
	Name           *string
	Email          *string
	Phone          *string
	Message        *string
	Source         *string
	OrganizationID int64
}

type ListLeadsResult struct {
	Items []*domain.Lead
	Total int64
}

func (s *LeadService) ListLeads(
	ctx context.Context,
	input ListLeadsInput,
) (*repository.ListLeadsResult, error) {
	if input.Limit <= 0 {
		input.Limit = 50
	}

	if input.Limit > 200 {
		input.Limit = 200
	}

	params := repository.ListLeadsParams{
		Limit:  input.Limit,
		Offset: input.Offset,
	}

	return s.repo.ListLeads(ctx, params)
}

func (s *LeadService) CreateLead(
	ctx context.Context,
	input CreateLeadInput,
	requestID string,
) (*domain.Lead, error) {

	existing, err := s.repo.FindByContact(
		ctx,
		input.OrganizationID,
		input.Email,
		input.Phone,
	)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		updated, err := s.UpdateLead(ctx, UpdateLeadInput{
			ID:             existing.ID,
			Name:           input.Name,
			Email:          input.Email,
			Phone:          input.Phone,
			Message:        input.Message,
			Source:         &input.Source,
			OrganizationID: input.OrganizationID,
		})
		if err != nil {
			return nil, err
		}

		if err := s.queue.PublishLeadCreated(
			ctx,
			updated.ID,
			requestID,
		); err != nil {
			slog.ErrorContext(
				ctx,
				"failed to publish lead created event",
				slog.String("request_id", requestID),
				slog.Int64("lead_id", updated.ID),
				slog.String("error", err.Error()),
			)
		}

		return updated, nil
	}

	params := repository.CreateLeadParams{
		Source:         input.Source,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		Message:        input.Message,
		OrganizationID: input.OrganizationID,
	}

	lead, err := s.repo.Create(ctx, params)
	if err != nil {
		return nil, err
	}

	if err := s.queue.PublishLeadCreated(
		ctx,
		lead.ID,
		requestID,
	); err != nil {
		slog.ErrorContext(
			ctx,
			"failed to publish lead created event",
			slog.String("request_id", requestID),
			slog.Int64("lead_id", lead.ID),
			slog.String("error", err.Error()),
		)
	}

	return lead, nil
}

func (s *LeadService) GetByID(
	ctx context.Context,
	id int64,
) (*domain.Lead, error) {
	if id <= 0 {
		return nil, domain.ErrLeadNotFound
	}

	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if lead == nil {
		return nil, domain.ErrLeadNotFound
	}

	return lead, nil

}

func (s *LeadService) UpdateLead(
	ctx context.Context,
	input UpdateLeadInput,
) (*domain.Lead, error) {

	if input.ID <= 0 {
		return nil, domain.ErrLeadNotFound
	}

	if input.Email != nil && *input.Email != "" {
		if _, err := mail.ParseAddress(*input.Email); err != nil {
			return nil, domain.ErrInvalidEmail
		}
	}

	params := repository.UpdateLeadParams{
		ID:             input.ID,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		Message:        input.Message,
		Source:         input.Source,
		OrganizationID: input.OrganizationID,
	}

	lead, err := s.repo.Update(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLeadNotFound
		}

		return nil, err
	}

	return lead, nil
}

func (s *LeadService) DeleteLead(
	ctx context.Context,
	id int64,
	organizationID int64,
) error {
	if id <= 0 {
		return domain.ErrLeadNotFound
	}

	return s.repo.Delete(ctx, id, organizationID)
}

func (s *LeadService) UpdateStatus(
	ctx context.Context,
	id int64,
	organizationID int64,
	status domain.LeadStatus,
) (*domain.Lead, error) {
	if id <= 0 {
		return nil, domain.ErrLeadNotFound
	}

	return s.repo.UpdateStatus(ctx, id, organizationID, status)
}

type ListLeadsInput struct {
	OrganizationID int64
	Status         *domain.LeadStatus
	Source         *string
	Search         string
	Sort           string
	Limit          int
	Offset         int
}

func (s *LeadService) ListForOrganization(
	ctx context.Context,
	input ListLeadsInput,
) ([]domain.Lead, int, error) {
	if input.Limit <= 0 {
		input.Limit = 50
	}

	if input.Limit > 100 {
		input.Limit = 100
	}

	if input.Offset < 0 {
		input.Offset = 0
	}

	if input.Sort != "created_at_asc" {
		input.Sort = "created_at_desc"
	}

	return s.repo.ListForOrganization(
		ctx,
		input.OrganizationID,
		input.Status,
		input.Source,
		input.Search,
		input.Sort,
		input.Limit,
		input.Offset,
	)
}
