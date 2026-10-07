package domain

import "time"

type Organization struct {
	ID               int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Name             string
	TelegramBotToken string
	TelegramChatID   int64
}
