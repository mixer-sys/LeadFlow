-- +goose Up

ALTER TABLE leads
ADD COLUMN IF NOT EXISTS organization_id BIGINT REFERENCES organizations(id);

-- +goose Down

ALTER TABLE leads
DROP COLUMN IF EXISTS organization_id;