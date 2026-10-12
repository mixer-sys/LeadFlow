-- +goose Up
ALTER TABLE leads
    ADD COLUMN bitrix_id BIGINT;

-- +goose Down
ALTER TABLE leads
    DROP COLUMN bitrix_id;
