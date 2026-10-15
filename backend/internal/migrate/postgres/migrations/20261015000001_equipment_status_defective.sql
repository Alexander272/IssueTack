-- +goose Up
-- +goose StatementBegin

-- Дефектная единица: неисправна, но ещё не списана и не отдана в ремонт.
-- Значение дописывается в enum (PG 12+ разрешает ADD VALUE в транзакции, если
-- новое значение не используется в этой же транзакции), поэтому его можно
-- выставлять сразу после применения миграции.
ALTER TYPE equipment_status ADD VALUE IF NOT EXISTS 'defective';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Удалить значение enum на месте нельзя — пересоздаём тип без 'defective',
-- а уже проставленные дефектные записи уводим в 'repair', чтобы не потерять их.
ALTER TABLE IF EXISTS public.equipment ALTER COLUMN status DROP DEFAULT;
ALTER TYPE equipment_status RENAME TO equipment_status_old;
CREATE TYPE equipment_status AS ENUM ('in_stock', 'in_use', 'repair', 'reserved', 'written_off');
ALTER TABLE IF EXISTS public.equipment
    ALTER COLUMN status TYPE equipment_status
    USING (CASE WHEN status::text = 'defective' THEN 'repair' ELSE status::text END)::equipment_status;
ALTER TABLE IF EXISTS public.equipment ALTER COLUMN status SET DEFAULT 'in_stock';
DROP TYPE IF EXISTS equipment_status_old;

-- +goose StatementEnd
