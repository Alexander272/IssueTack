-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.role_permissions (
    role_id        UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id  UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,

    PRIMARY KEY (role_id, permission_id)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.role_permissions
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.role_permissions;
-- +goose StatementEnd
