-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS public.tickets
    ADD COLUMN IF NOT EXISTS owner_muted BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE IF EXISTS public.tickets
    DROP COLUMN IF EXISTS owner_muted;
-- +goose StatementEnd
