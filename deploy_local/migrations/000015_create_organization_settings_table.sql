-- +goose Up
CREATE TABLE organization_settings (
    organization_id BIGINT PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    telegram_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    telegram_bot_token_encrypted TEXT NOT NULL DEFAULT '',
    telegram_chat_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS organization_settings;
