-- +goose Up
CREATE TABLE telegram_channels (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    bot_token TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_telegram_channels_organization_id
ON telegram_channels(organization_id);

-- +goose Down
DROP INDEX IF EXISTS idx_telegram_channels_organization_id;
DROP TABLE IF EXISTS organizations;
