package service

import (
	"context"

	"leadflow/internal/domain"
	"leadflow/internal/repository"
)

type OrganizationService struct {
	repo *repository.OrganizationRepo
}

func NewOrganizationService(repo *repository.OrganizationRepo) *OrganizationService {
	return &OrganizationService{repo: repo}
}

type CreateOrganizationInput struct {
	Name             string
	TelegramBotToken string
	TelegramChatID   int64
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, input CreateOrganizationInput) (*domain.Organization, error) {
	params := repository.CreateOrganizationParams{
		Name:             input.Name,
		TelegramBotToken: input.TelegramBotToken,
		TelegramChatID:   input.TelegramChatID,
	}

	return s.repo.Create(ctx, params)
}
