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
