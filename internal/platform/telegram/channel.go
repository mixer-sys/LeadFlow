package telegram

import (
	"context"
	"time"
)

type Channel struct {
	ID             int64
	OrganizationID int64
	Name           string
	BotToken       string
	ChatID         string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ChannelRepository interface {
	List(ctx context.Context, organizationID int64) ([]Channel, error)
	ListActive(ctx context.Context, organizationID int64) ([]Channel, error)
	Create(ctx context.Context, channel Channel) (Channel, error)
	Get(ctx context.Context, organizationID, id int64) (Channel, error)
	Toggle(ctx context.Context, organizationID, id int64) error
	Delete(ctx context.Context, organizationID, id int64) error
}
