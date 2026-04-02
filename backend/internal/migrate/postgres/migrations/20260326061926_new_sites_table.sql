-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.sites
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text COLLATE pg_catalog."default" NOT NULL,
    address text COLLATE pg_catalog."default" DEFAULT ''::text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.sites
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.sites;
-- +goose StatementEnd
