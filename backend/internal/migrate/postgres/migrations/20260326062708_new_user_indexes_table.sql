-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_roles_realm ON roles(realm_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_roles_realm;
DROP INDEX IF EXISTS idx_user_roles_user;
DROP INDEX IF EXISTS idx_role_permissions_role;
-- +goose StatementEnd
