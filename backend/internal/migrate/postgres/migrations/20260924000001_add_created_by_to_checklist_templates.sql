-- +goose Up
-- +goose StatementBegin
ALTER TABLE public.checklist_templates ADD COLUMN IF NOT EXISTS created_by UUID NULL REFERENCES public.users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_checklist_templates_created_by ON public.checklist_templates(created_by);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_checklist_templates_created_by;
ALTER TABLE public.checklist_templates DROP COLUMN IF EXISTS created_by;
-- +goose StatementEnd