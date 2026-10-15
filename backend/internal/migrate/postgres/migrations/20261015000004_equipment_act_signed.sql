-- +goose Up
-- +goose StatementBegin

-- Жизненный цикл акта: сформирован (docx отдан пользователю) → подписан
-- (подгружен скан/фото подписанного документа). Файл подписанного акта лежит на
-- диске (uploads/equipment_act/<act_id>/), в БД — только его реквизиты и путь;
-- снимок состава акта остаётся единственным источником документа, поэтому docx
-- не хранится, а собирается по запросу.
ALTER TABLE public.equipment_acts
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'formed',
    ADD COLUMN IF NOT EXISTS signed_file_name TEXT,
    ADD COLUMN IF NOT EXISTS signed_file_path TEXT,
    ADD COLUMN IF NOT EXISTS signed_mime_type TEXT,
    ADD COLUMN IF NOT EXISTS signed_file_size BIGINT,
    ADD COLUMN IF NOT EXISTS signed_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS signed_by UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE public.equipment_acts DROP CONSTRAINT IF EXISTS equipment_acts_status_check;
ALTER TABLE public.equipment_acts
    ADD CONSTRAINT equipment_acts_status_check CHECK (status IN ('formed', 'signed'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE public.equipment_acts DROP CONSTRAINT IF EXISTS equipment_acts_status_check;
ALTER TABLE public.equipment_acts
    DROP COLUMN IF EXISTS signed_by,
    DROP COLUMN IF EXISTS signed_at,
    DROP COLUMN IF EXISTS signed_file_size,
    DROP COLUMN IF EXISTS signed_mime_type,
    DROP COLUMN IF EXISTS signed_file_path,
    DROP COLUMN IF EXISTS signed_file_name,
    DROP COLUMN IF EXISTS status;

-- +goose StatementEnd
