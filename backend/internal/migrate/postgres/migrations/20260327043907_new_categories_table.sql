-- +goose Up
-- +goose StatementBegin
DO $$ 
BEGIN 
    CREATE TYPE ticket_priority AS ENUM ('low', 'medium', 'high', 'urgent');
EXCEPTION 
    WHEN duplicate_object THEN NULL; 
END $$;

CREATE TABLE IF NOT EXISTS public.categories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id     UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name         TEXT COLLATE pg_catalog."default" NOT NULL,
    description  TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    group_id     UUID REFERENCES groups(id) ON DELETE SET NULL,
    def_priority ticket_priority NOT NULL DEFAULT 'medium',
    is_active    BOOLEAN DEFAULT TRUE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(realm_id, name)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.categories
    OWNER to postgres;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.categories;

DROP TYPE IF EXISTS ticket_priority;
-- +goose StatementEnd
