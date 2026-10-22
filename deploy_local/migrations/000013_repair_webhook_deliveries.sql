-- +goose Up

-- Создаём таблицу, если её нет.
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id BIGSERIAL PRIMARY KEY,
    webhook_id BIGINT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    lead_id BIGINT NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (webhook_id, lead_id, event_id)
);

-- Добавляем недостающие поля, если таблица уже существовала.
ALTER TABLE webhook_deliveries
    ADD COLUMN IF NOT EXISTS request_id TEXT NOT NULL DEFAULT '';

ALTER TABLE webhook_deliveries
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

ALTER TABLE webhook_deliveries
    ADD COLUMN IF NOT EXISTS max_attempts INT NOT NULL DEFAULT 5;

-- Индекс для выборки webhooks, готовых к retry.
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_due
    ON webhook_deliveries (status, next_retry_at)
    WHERE status IN ('pending', 'failed');

-- +goose Down
DROP TABLE IF EXISTS webhook_deliveries;
