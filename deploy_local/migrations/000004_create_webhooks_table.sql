-- +goose Up

CREATE TABLE IF NOT EXISTS webhooks (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_webhooks_is_active
    ON webhooks (is_active);

-- +goose Down

DROP INDEX IF EXISTS idx_webhooks_is_active;
DROP TABLE IF EXISTS webhooks;