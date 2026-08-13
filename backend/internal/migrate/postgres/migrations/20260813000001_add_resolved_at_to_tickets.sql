-- +goose Up
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.tickets
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP WITH TIME ZONE;

-- Для уже решённых тикетов берём время закрытия (или последнего обновления)
UPDATE public.tickets
SET resolved_at = COALESCE(closed_at, updated_at)
WHERE status = 'resolved' AND resolved_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.tickets
    DROP COLUMN IF EXISTS resolved_at;

-- +goose StatementEnd
