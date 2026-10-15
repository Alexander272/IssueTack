-- +goose Up
-- +goose StatementBegin

-- Стоимость оборудования в программе не хранится — убираем колонку.
ALTER TABLE IF EXISTS public.equipment DROP COLUMN IF EXISTS cost;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.equipment ADD COLUMN IF NOT EXISTS cost NUMERIC(12, 2);

-- +goose StatementEnd
