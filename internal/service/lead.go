package service

import (
	"context"

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
	Source  string
	Name    *string
	Email   *string
	Phone   *string
	Message *string
}

func (s *LeadService) CreateLead(ctx context.Context, input CreateLeadInput) (*domain.Lead, error) {
	params := repository.CreateLeadParams{
		Source:  input.Source,
		Name:    input.Name,
		Email:   input.Email,
		Phone:   input.Phone,
		Message: input.Message,
	}

	lead, err := s.repo.Create(ctx, params)
	if err != nil {
		return nil, err
	}

	if err := s.queue.PublishLeadCreated(
		ctx, lead.ID,
	); err != nil {
		// logging
	}

	return lead, nil
}
