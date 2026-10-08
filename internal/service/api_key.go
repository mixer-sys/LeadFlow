package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"leadflow/internal/domain"
	"leadflow/internal/repository"
)

type APIKeyService struct {
	repo *repository.APIKeyRepo
}

func NewAPIKeyService(repo *repository.APIKeyRepo) *APIKeyService {
	return &APIKeyService{repo: repo}
}

func (s *APIKeyService) GenerateKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (s *APIKeyService) CreateAPIKey(ctx context.Context, name string) (string, error) {
	key, err := s.GenerateKey()
	if err != nil {
		return "", err
	}

	_, err = s.repo.Create(ctx, key, name)
	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *APIKeyService) ValidateKey(ctx context.Context, key string) (bool, error) {
	name, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return false, err
	}
	return name != nil, nil
}

func (s *APIKeyService) ListAPIKeys(ctx context.Context) ([]domain.APIKey, error) {
	return s.repo.List(ctx)
}

func (s *APIKeyService) DeleteAPIKey(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *APIKeyService) ToggleWebhookActive(ctx context.Context, id int64, isActive bool) error {
	return s.repo.ToggleActive(ctx, id, isActive)
}
