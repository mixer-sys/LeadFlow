-- +goose Up
ALTER TABLE webhooks
ADD COLUMN organization_id BIGINT,
ADD COLUMN secret TEXT NOT NULL DEFAULT '',
ADD COLUMN events TEXT NOT NULL DEFAULT 'lead.created';

CREATE INDEX idx_webhooks_organization_id
ON webhooks(organization_id);

-- +goose Down
DROP INDEX IF EXISTS idx_webhooks_organization_id;

ALTER TABLE webhooks
DROP COLUMN IF EXISTS organization_id,
DROP COLUMN IF EXISTS secret,
DROP COLUMN IF EXISTS events;