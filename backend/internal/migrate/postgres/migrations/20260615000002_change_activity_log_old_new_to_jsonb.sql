-- +goose Up
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.activity_logs
    ALTER COLUMN old_value TYPE JSONB
    USING old_value::jsonb;

ALTER TABLE IF EXISTS public.activity_logs
    ALTER COLUMN new_value TYPE JSONB
    USING new_value::jsonb;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.activity_logs
    ALTER COLUMN new_value TYPE TEXT
    USING CASE WHEN new_value IS NULL THEN NULL ELSE new_value::text END;

ALTER TABLE IF EXISTS public.activity_logs
    ALTER COLUMN old_value TYPE TEXT
    USING CASE WHEN old_value IS NULL THEN NULL ELSE old_value::text END;

-- +goose StatementEnd
