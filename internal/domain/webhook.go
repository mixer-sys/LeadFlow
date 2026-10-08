package domain

import "time"

type Webhook struct {
	ID        int64
	URL       string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
