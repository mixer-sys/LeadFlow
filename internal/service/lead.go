package service

import (
	"context"
	"fmt"
	"os"

	"leadflow/internal/domain"
	"leadflow/internal/platform/queue"
	"leadflow/internal/repository"
)

type LeadService struct {
	repo  *repository.LeadRepo
	queue *queue.RedisQueue
}

func NewLeadService(repo *repository.LeadRepo, q *queue.RedisQueue) *LeadService {
	return &LeadService{repo: repo, queue: q}
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

type ListLeadsInput struct {
	Limit  int
	Offset int
}

type ListLeadsResult struct {
	Items []*domain.Lead
	Total int64
}

func (s *LeadService) ListLeads(ctx context.Context, input ListLeadsInput) (*repository.ListLeadsResult, error) {
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

func (s *LeadService) CreateLead(ctx context.Context, input CreateLeadInput) (*domain.Lead, error) {
	fmt.Fprintf(os.Stderr, "Before CreateLeadParams: org_id=%d\n", input.OrganizationID)

	existing, err := s.repo.FindByContact(ctx, input.OrganizationID, input.Email, input.Phone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FindByContact error: %v\n", err)
		return nil, err
	}

	if existing != nil {
		fmt.Fprintf(os.Stderr, "DUPLICATE FOUND: lead_id=%d, email=%v, phone=%v\n", existing.ID, input.Email, input.Phone)

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
			fmt.Fprintf(os.Stderr, "UpdateLead error: %v\n", err)
			return nil, err
		}

		if err := s.queue.PublishLeadCreated(ctx, updated.ID); err != nil {
			fmt.Fprintf(os.Stderr, "queue.PublishLeadCreated error: %v\n", err)
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

	fmt.Fprintf(os.Stderr, "After CreateLeadParams: org_id=%d\n", params.OrganizationID)

	lead, err := s.repo.Create(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repo.Create error: %v\n", err)
		return nil, err
	}

	if err := s.queue.PublishLeadCreated(
		ctx, lead.ID,
	); err != nil {
		// logging
		fmt.Fprintf(os.Stderr, "queue.PublishLeadCreated error: %v\n", err)
	}

	return lead, nil
}

func (s *LeadService) UpdateLead(ctx context.Context, input UpdateLeadInput) (*domain.Lead, error) {
	params := repository.UpdateLeadParams{
		ID:             input.ID,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		Message:        input.Message,
		Source:         input.Source,
		OrganizationID: input.OrganizationID,
	}

	return s.repo.Update(ctx, params)
}
