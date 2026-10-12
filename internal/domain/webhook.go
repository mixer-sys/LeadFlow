package domain

import "time"

type WebhookDeliveryStatus string

const (
	WebhookDeliveryPending   WebhookDeliveryStatus = "pending"
	WebhookDeliveryDelivered WebhookDeliveryStatus = "delivered"
	WebhookDeliveryFailed    WebhookDeliveryStatus = "failed"
)

type WebhookDelivery struct {
	ID          int64
	WebhookID   int64
	LeadID      int64
	EventID     string
	Status      WebhookDeliveryStatus
	Attempts    int
	LastError   *string
	NextRetryAt *time.Time
	MaxAttempts int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Webhook struct {
	ID           int64
	URL          string
	IsActive     bool
	FailureCount int
	OpenedUntil  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
