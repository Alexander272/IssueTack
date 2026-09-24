-- +goose Up
-- +goose StatementBegin

ALTER TABLE public.subtasks
    ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- Бэкфилл: для существующих подзадач берём первого автора из журнала
-- (запись 'created' по entity_type='subtask').
UPDATE public.subtasks AS s
SET created_by = al.changed_by
FROM (
    SELECT DISTINCT ON (entity_id) entity_id, changed_by
    FROM public.activity_logs
    WHERE entity_type = 'subtask' AND action = 'created'
    ORDER BY entity_id, created_at
) AS al
WHERE al.entity_id = s.id AND s.created_by IS NULL;

CREATE INDEX IF NOT EXISTS idx_subtasks_created_by ON public.subtasks(created_by);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_subtasks_created_by;
ALTER TABLE public.subtasks DROP COLUMN IF EXISTS created_by;

-- +goose StatementEnd