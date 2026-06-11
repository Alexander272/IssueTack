-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.realms
(
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    code TEXT COLLATE pg_catalog."default" NOT NULL UNIQUE,
    name TEXT COLLATE pg_catalog."default" NOT NULL,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.realms
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.realms;
-- +goose StatementEnd
