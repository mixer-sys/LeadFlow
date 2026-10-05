package domain

import "time"

type LeadStatus string

const (
	LeadStatusNew       LeadStatus = "new"
	LeadStatusProcessed LeadStatus = "processed"
	LeadStatusFailed    LeadStatus = "failed"
)

type Lead struct {
	ID                int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Source            string
	Name              *string
	Email             *string
	Phone             *string
	Message           *string
	Status            LeadStatus
	ProcessedAt       *time.Time
	TelegramMessageID *int64
	TelegramSentAt    *time.Time
}
