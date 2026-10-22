-- +goose Up
ALTER TABLE webhook_deliveries
ADD COLUMN IF NOT EXISTS request_id TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE webhook_deliveries
DROP COLUMN IF EXISTS request_id;

