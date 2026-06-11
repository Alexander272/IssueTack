-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.users
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sso_id text COLLATE pg_catalog."default" NOT NULL,
    mattermost_id text COLLATE pg_catalog."default" DEFAULT ''::text,
    username text COLLATE pg_catalog."default" DEFAULT ''::text,
    first_name text COLLATE pg_catalog."default" DEFAULT ''::text,
    last_name text COLLATE pg_catalog."default" DEFAULT ''::text,
    email text COLLATE pg_catalog."default" DEFAULT ''::text,
    site_id uuid REFERENCES sites(id) ON DELETE SET NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.users
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.users;
-- +goose StatementEnd
