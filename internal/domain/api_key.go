package domain

import "time"

type APIKey struct {
	ID        int64
	Key       string
	Name      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
