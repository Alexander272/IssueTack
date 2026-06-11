-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.activity_logs (
    id UUID PRIMARY KEY,
    action TEXT NOT NULL,                                   -- 'INSERT', 'UPDATE', 'DELETE'
    changed_by UUID NOT NULL,                               -- Кто изменил (UserID)
    changed_by_name TEXT NOT NULL,

    entity_type TEXT NOT NULL,          -- 'ticket', 'comment', 'group', 'user', etc.
    entity_id UUID NOT NULL,            -- ID сущности
    entity TEXT DEFAULT ''::text,       -- отображаемое имя сущности (напр. заголовок тикета)
    parent_id UUID,                     -- ID родительской сущности (для вложенных)

    realm_id UUID,
    realm_name TEXT,                    -- название realm на момент события (денормализация)

    old_value TEXT,
    new_value TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

ALTER TABLE IF EXISTS public.activity_logs
    OWNER to postgres;

CREATE INDEX idx_activity_logs_entity ON activity_logs(entity_type, entity_id);
CREATE INDEX idx_activity_logs_parent ON activity_logs(parent_id);
CREATE INDEX idx_activity_logs_realm ON activity_logs(realm_id);
CREATE INDEX idx_activity_logs_created_at ON activity_logs(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_activity_logs_entity;
DROP INDEX IF EXISTS idx_activity_logs_parent;
DROP INDEX IF EXISTS idx_activity_logs_realm;
DROP INDEX IF EXISTS idx_activity_logs_created_at;

DROP TABLE IF EXISTS public.activity_logs;
-- +goose StatementEnd
