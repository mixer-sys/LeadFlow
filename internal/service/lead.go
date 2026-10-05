package service

import (
	"context"

	"leadflow/internal/domain"
	"leadflow/internal/repository"
)

type LeadService struct {
	repo *repository.LeadRepo
}

func NewLeadService(repo *repository.LeadRepo) *LeadService {
	return &LeadService{repo: repo}
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

	return s.repo.Create(ctx, params)
}
