-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.roles (
    id          uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    slug        TEXT COLLATE pg_catalog."default" NOT NULL,
    name        TEXT COLLATE pg_catalog."default" NOT NULL,
    realm_id   uuid NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    level       INT DEFAULT 1,
    is_system   BOOLEAN DEFAULT false,
    created_at  TIMESTAMP with time zone DEFAULT now(),
    updated_at  TIMESTAMP with time zone DEFAULT now(),

    UNIQUE(slug, realm_id)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.roles
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.roles;
-- +goose StatementEnd
