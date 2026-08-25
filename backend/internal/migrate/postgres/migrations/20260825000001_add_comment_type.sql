-- +goose Up

ALTER TABLE comments ADD COLUMN type VARCHAR(32) NOT NULL DEFAULT 'user';

-- +goose Down

ALTER TABLE comments DROP COLUMN IF EXISTS type;
