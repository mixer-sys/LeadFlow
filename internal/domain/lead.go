package domain

import (
	"time"
)

type LeadStatus string

const (
	LeadStatusNew       LeadStatus = "new"
	LeadStatusProcessed LeadStatus = "processed"
	LeadStatusFailed    LeadStatus = "failed"
)

type Lead struct {
	ID                 int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Source             string
	Name               *string
	Email              *string
	Phone              *string
	Message            *string
	Status             LeadStatus
	ProcessedAt        *time.Time
	TelegramMessageID  *int64
	TelegramSentAt     *time.Time
	TelegramRetryCount int
	OrganizationID     int64
	BitrixID           *int64
}

type CreateLeadParams struct {
	Source         string
	Name           *string
	Email          *string
	Phone          *string
	Message        *string
	OrganizationID int64
}

func (s LeadStatus) IsValid() bool {
	switch s {
	case LeadStatusFailed, LeadStatusNew, LeadStatusProcessed:
		return true
	default:
		return false
	}
}
