-- +goose Up

CREATE TABLE IF NOT EXISTS organizations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    name TEXT NOT NULL,
    telegram_bot_token TEXT,
    telegram_chat_id BIGINT
);

-- +goose Down

DROP TABLE IF EXISTS organizations;