-- +goose Up
ALTER TABLE leads
  ADD COLUMN IF NOT EXISTS telegram_retry_count integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE leads
  DROP COLUMN IF EXISTS telegram_retry_count;
