package domain

import (
	"time"
)

type TelegramSettings struct {
	OrganizationID    int64
	Enabled           bool
	BotTokenEncrypted string
	ChatID            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
