-- +goose Up
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.users
    ADD COLUMN IF NOT EXISTS internal_number text DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.users
    DROP COLUMN IF EXISTS internal_number;

-- +goose StatementEnd
