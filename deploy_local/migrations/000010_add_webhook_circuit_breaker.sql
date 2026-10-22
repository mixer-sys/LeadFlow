-- +goose Up
ALTER TABLE webhooks
    ADD COLUMN failure_count INT NOT NULL DEFAULT 0,
    ADD COLUMN opened_until TIMESTAMPTZ;

-- +goose Down
ALTER TABLE webhooks
    DROP COLUMN IF EXISTS failure_count,
    DROP COLUMN IF EXISTS opened_until;
