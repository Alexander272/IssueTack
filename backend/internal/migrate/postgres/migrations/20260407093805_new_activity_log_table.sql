-- +goose Up
-- +goose StatementBegin
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.activity_logs (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL,
    user_id UUID NOT NULL,
    type TEXT COLLATE pg_catalog."default" NOT NULL,
    
    old_value TEXT COLLATE pg_catalog."default",
    new_value TEXT COLLATE pg_catalog."default",
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.activity_logs
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.activity_logs;
-- +goose StatementEnd
