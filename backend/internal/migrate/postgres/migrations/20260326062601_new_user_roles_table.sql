-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,

    PRIMARY KEY (user_id, role_id)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.user_roles
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.user_roles;
-- +goose StatementEnd
