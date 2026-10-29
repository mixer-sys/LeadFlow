package service

import (
	"context"
	"fmt"

	"leadflow/internal/platform/crypto"
	"leadflow/internal/platform/telegram"
)

type TelegramChannelService struct {
	repo      telegram.ChannelRepository
	encryptor *crypto.Encryptor
	sender    *telegram.SenderAdapter
}

func NewTelegramChannelService(
	repo telegram.ChannelRepository,
	encryptor *crypto.Encryptor,
	sender *telegram.SenderAdapter,
) *TelegramChannelService {
	return &TelegramChannelService{
		repo:      repo,
		encryptor: encryptor,
		sender:    sender,
	}
}

func (s *TelegramChannelService) List(
	ctx context.Context,
	organizationID int64,
) ([]telegram.Channel, error) {
	return s.repo.List(ctx, organizationID)
}

func (s *TelegramChannelService) ListActive(
	ctx context.Context,
	organizationID int64,
) ([]telegram.Channel, error) {
	return s.repo.ListActive(ctx, organizationID)
}

func (s *TelegramChannelService) Create(
	ctx context.Context,
	channel telegram.Channel,
) (telegram.Channel, error) {
	if channel.Name == "" {
		return telegram.Channel{}, fmt.Errorf("name is required")
	}

	if channel.BotToken == "" {
		return telegram.Channel{}, fmt.Errorf("bot token is required")
	}

	if channel.ChatID == "" {
		return telegram.Channel{}, fmt.Errorf("chat id is required")
	}

	return s.repo.Create(ctx, channel)
}

func (s *TelegramChannelService) Get(
	ctx context.Context,
	organizationID, id int64,
) (telegram.Channel, error) {
	return s.repo.Get(ctx, organizationID, id)
}

func (s *TelegramChannelService) Toggle(
	ctx context.Context,
	organizationID, id int64,
) error {
	return s.repo.Toggle(ctx, organizationID, id)
}

func (s *TelegramChannelService) Delete(
	ctx context.Context,
	organizationID, id int64,
) error {
	return s.repo.Delete(ctx, organizationID, id)
}

func (s *TelegramChannelService) SendMessage(
	ctx context.Context,
	botToken, chatID, text string,
) (int64, error) {
	return s.sender.SendMessage(ctx, botToken, chatID, text)
}
