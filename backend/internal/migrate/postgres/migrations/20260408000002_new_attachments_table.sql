-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.attachments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type TEXT NOT NULL,           -- 'ticket' | 'subtask'
    entity_id   UUID NOT NULL,
    file_name   TEXT NOT NULL,
    file_path   TEXT NOT NULL,           -- путь на диске
    file_size   BIGINT DEFAULT 0,
    mime_type   TEXT DEFAULT ''::text,
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT now()
);

ALTER TABLE IF EXISTS public.attachments
    OWNER to postgres;

CREATE INDEX idx_attachments_entity ON attachments(entity_type, entity_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_attachments_entity;
DROP TABLE IF EXISTS public.attachments;
-- +goose StatementEnd
