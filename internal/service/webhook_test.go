package service

import (
	"testing"
	"time"
)

func TestWebhookRetryDelay(t *testing.T) {
	delay := webhookRetryDelay(1)

	if delay <= 0 {
		t.Fatalf("expected positive delay, got %s", delay)
	}

	if delay > 5*time.Minute {
		t.Fatalf("expected delay <= 5m, got %s", delay)
	}
}

func TestWebhookOpenDuration(t *testing.T) {
	tests := []struct {
		name         string
		failureCount int
		want         time.Duration
	}{
		{
			name:         "below threshold",
			failureCount: 4,
			want:         0,
		},
		{
			name:         "first open",
			failureCount: 5,
			want:         30 * time.Second,
		},
		{
			name:         "second open",
			failureCount: 6,
			want:         60 * time.Second,
		},
		{
			name:         "capped at maximum",
			failureCount: 20,
			want:         10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := webhookOpenDuration(tt.failureCount)

			if got != tt.want {
				t.Fatalf(
					"webhookOpenDuration(%d) = %s, want%s", tt.failureCount, got,
					tt.want,
				)
			}
		})
	}
}
