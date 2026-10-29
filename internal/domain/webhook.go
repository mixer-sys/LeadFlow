package domain

import (
	"errors"
	"time"
)

type WebhookDeliveryStatus string

const (
	WebhookDeliveryPending   WebhookDeliveryStatus = "pending"
	WebhookDeliveryDelivered WebhookDeliveryStatus = "delivered"
	WebhookDeliveryFailed    WebhookDeliveryStatus = "failed"
)

var ErrWebhookNotFound = errors.New("webhook not found")

type WebhookDelivery struct {
	ID          int64
	WebhookID   int64
	LeadID      int64
	EventID     string
	RequestID   string
	Status      WebhookDeliveryStatus
	Attempts    int
	LastError   *string
	NextRetryAt *time.Time
	MaxAttempts int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Webhook struct {
	ID             int64
	OrganizationID int64
	URL            string
	Secret         string
	Events         string
	IsActive       bool
	FailureCount   int
	OpenedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
