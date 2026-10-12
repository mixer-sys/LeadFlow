-- +goose Up
ALTER TABLE api_keys
    ADD COLUMN organization_id BIGINT NOT NULL DEFAULT 1;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_organization_id_fkey
    FOREIGN KEY (organization_id)
    REFERENCES organizations(id);

-- +goose Down
ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_organization_id_fkey;

ALTER TABLE api_keys
    DROP COLUMN IF EXISTS organization_id;