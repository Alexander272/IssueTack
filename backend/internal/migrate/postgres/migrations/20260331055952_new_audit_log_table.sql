-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.policy_audit_logs (
    id UUID PRIMARY KEY,
    changed_by UUID NOT NULL,     -- кто изменил (user_id)
    changed_by_name TEXT COLLATE pg_catalog."default" NOT NULL,
    action TEXT COLLATE pg_catalog."default" NOT NULL,          -- "add_role", "remove_permission", etc.
    
    entity_type TEXT COLLATE pg_catalog."default" NOT NULL,     -- 'user' или 'role'
    entity TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    entity_id UUID, 

    realm_name TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    realm_id UUID, 
    
    old_values JSONB,
    new_values JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.policy_audit_logs
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.policy_audit_logs;
-- +goose StatementEnd
