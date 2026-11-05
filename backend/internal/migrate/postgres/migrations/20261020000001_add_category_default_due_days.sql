-- +goose Up
ALTER TABLE categories ADD COLUMN IF NOT EXISTS default_due_days INT NULL;

-- +goose Down
ALTER TABLE categories DROP COLUMN IF EXISTS default_due_days;