-- +goose Up
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.users
    ADD COLUMN IF NOT EXISTS is_active boolean DEFAULT true;

ALTER TABLE IF EXISTS public.realms
    ADD COLUMN IF NOT EXISTS is_active boolean DEFAULT true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.realms
    DROP COLUMN IF EXISTS is_active;

ALTER TABLE IF EXISTS public.users
    DROP COLUMN IF EXISTS is_active;

-- +goose StatementEnd
