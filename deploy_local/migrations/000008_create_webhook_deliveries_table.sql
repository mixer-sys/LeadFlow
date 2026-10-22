-- +goose Up
CREATE TABLE webhook_deliveries (
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

CREATE INDEX idx_webhook_deliveries_lead_id
    ON webhook_deliveries (lead_id);

-- +goose Down
DROP TABLE IF EXISTS webhook_deliveries;