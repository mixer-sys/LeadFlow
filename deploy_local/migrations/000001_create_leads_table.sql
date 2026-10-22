-- +goose Up

CREATE TABLE IF NOT EXISTS leads (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    source TEXT NOT NULL,
    name TEXT,
    email TEXT,
    phone TEXT,
    message TEXT,

    status TEXT NOT NULL DEFAULT 'new',
    processed_at TIMESTAMPTZ,
    telegram_message_id BIGINT,
    telegram_sent_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_leads_created_at
    ON leads (created_at);

CREATE INDEX IF NOT EXISTS idx_leads_status
    ON leads (status);

-- +goose Down

DROP INDEX IF EXISTS idx_leads_status;
DROP INDEX IF EXISTS idx_leads_created_at;
DROP TABLE IF EXISTS leads;