-- +goose Up
-- +goose StatementBegin

-- Явная иконка типа: ключ из закрытого набора фронтового реестра
-- (frontend .../equipment/typeIcons.tsx). Пустая строка = «Авто»: фронт
-- подбирает иконку по названию типа, поэтому существующие типы не требуют
-- бэкфилла — они продолжают отображаться как раньше.
ALTER TABLE public.equipment_types
    ADD COLUMN IF NOT EXISTS icon TEXT NOT NULL DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Откат просто убирает колонку: никакой логики, кроме косметики иконок, она
-- не несёт.
ALTER TABLE public.equipment_types
    DROP COLUMN IF EXISTS icon;

-- +goose StatementEnd
