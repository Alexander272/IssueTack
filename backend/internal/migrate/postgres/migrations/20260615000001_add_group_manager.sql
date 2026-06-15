-- +goose Up
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.groups
    ADD COLUMN IF NOT EXISTS default_assignee_id UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE IF EXISTS public.groups
    ADD COLUMN IF NOT EXISTS manager_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.groups
    DROP COLUMN IF EXISTS manager_id;

ALTER TABLE IF EXISTS public.groups
    DROP COLUMN IF EXISTS default_assignee_id;

-- +goose StatementEnd
