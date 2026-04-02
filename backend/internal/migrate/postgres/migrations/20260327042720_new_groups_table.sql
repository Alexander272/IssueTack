-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id    UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name        TEXT COLLATE pg_catalog."default" NOT NULL,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(realm_id, name)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.groups
    OWNER to postgres;

CREATE TABLE IF NOT EXISTS public.group_members (
    group_id UUID REFERENCES groups(id) ON DELETE CASCADE,
    user_id  UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.groups
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.group_members;
DROP TABLE IF EXISTS public.groups;
-- +goose StatementEnd
