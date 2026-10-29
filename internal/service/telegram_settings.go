package service

import (
	"context"
	"errors"

	"leadflow/internal/domain"
	"leadflow/internal/platform/crypto"
)

type TelegramSettingsRepo interface {
	Get(
		ctx context.Context,
		organizationID int64,
	) (*domain.TelegramSettings, error)

	Save(
		ctx context.Context,
		settings *domain.TelegramSettings,
	) error
}

type TelegramSender interface {
	SendMessage(
		ctx context.Context,
		botToken string,
		chatID string,
		text string,
	) (int64, error)
}

type TelegramSettingsService struct {
	repo      TelegramSettingsRepo
	encryptor *crypto.Encryptor
	sender    TelegramSender
}

func NewTelegramSettingsService(
	repo TelegramSettingsRepo,
	encryptor *crypto.Encryptor,
	sender TelegramSender,
) *TelegramSettingsService {
	return &TelegramSettingsService{
		repo:      repo,
		encryptor: encryptor,
		sender:    sender,
	}
}

type SaveTelegramSettingsInput struct {
	OrganizationID int64
	Enabled        bool
	BotToken       string
	ChatID         string
}

func (s *TelegramSettingsService) Get(
	ctx context.Context,
	organizationID int64,
) (*domain.TelegramSettings, error) {
	return s.repo.Get(ctx, organizationID)
}

func (s *TelegramSettingsService) Save(
	ctx context.Context,
	input SaveTelegramSettingsInput,
) error {
	if input.ChatID == "" {
		return errors.New("telegram chat id is required")
	}

	settings, err := s.repo.Get(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	encryptedToken := settings.BotTokenEncrypted

	if input.BotToken != "" {
		encryptedToken, err = s.encryptor.Encrypt(input.BotToken)
		if err != nil {
			return err
		}
	}

	settings.Enabled = input.Enabled
	settings.ChatID = input.ChatID
	settings.BotTokenEncrypted = encryptedToken

	return s.repo.Save(ctx, settings)
}

func (s *TelegramSettingsService) SendTestMessage(
	ctx context.Context,
	organizationID int64,
) error {
	settings, err := s.repo.Get(ctx, organizationID)
	if err != nil {
		return err
	}

	if !settings.Enabled {
		return errors.New("telegram is disabled")
	}

	botToken, err := s.encryptor.Decrypt(
		settings.BotTokenEncrypted,
	)
	if err != nil {
		return err
	}

	if botToken == "" || settings.ChatID == "" {
		return errors.New("telegram is not configured")
	}

	_, err = s.sender.SendMessage(
		ctx,
		botToken,
		settings.ChatID,
		"LeadFlow: test notification",
	)

	return err
}

func (s *TelegramSettingsService) DecryptBotToken(
	ctx context.Context,
	settings *domain.TelegramSettings,
) (string, error) {
	return s.encryptor.Decrypt(settings.BotTokenEncrypted)
}

func (s *TelegramSettingsService) SendMessage(
	ctx context.Context,
	botToken string,
	chatID string,
	text string,
) (int64, error) {
	return s.sender.SendMessage(
		ctx,
		botToken,
		chatID,
		text,
	)
}
