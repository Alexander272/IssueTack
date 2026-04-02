-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.policy_audit_log (
    id UUID PRIMARY KEY,
    changed_by UUID NOT NULL,     -- кто изменил (user_id)
    action TEXT COLLATE pg_catalog."default" NOT NULL,          -- "add_role", "remove_permission", etc.
    
    role_id UUID,
    rule_id UUID,
    realm_id UUID,
    user_id UUID,                 -- если меняли назначение роли пользователю
    
    old_values JSONB,
    new_values JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.policy_audit_log
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.policy_audit_log;
-- +goose StatementEnd
