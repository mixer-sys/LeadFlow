-- +goose Up
ALTER TABLE webhook_deliveries
    ADD COLUMN next_retry_at TIMESTAMPTZ,
    ADD COLUMN max_attempts INT NOT NULL DEFAULT 5;

CREATE INDEX idx_webhook_deliveries_due
    ON webhook_deliveries (status, next_retry_at)
    WHERE status IN ('pending', 'failed');

-- +goose Down
DROP INDEX IF EXISTS idx_webhook_deliveries_due;

ALTER TABLE webhook_deliveries
    DROP COLUMN IF EXISTS next_retry_at,
    DROP COLUMN IF EXISTS max_attempts;
