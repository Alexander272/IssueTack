-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.permissions (
    id          UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    object      TEXT COLLATE pg_catalog."default" NOT NULL, -- task, user, document
    action      TEXT COLLATE pg_catalog."default" NOT NULL, -- read, write, delete
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,

    UNIQUE(object, action)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.permissions
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.permissions;
-- +goose StatementEnd
