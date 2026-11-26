-- +goose Up
-- +goose StatementBegin

-- Ключевые слова подкатегории: сервер выбирает модификатор по словам в
-- заголовке/описании новой заявки. Пустой массив — автоподбор не применяется.
ALTER TABLE public.category_modifiers
    ADD COLUMN IF NOT EXISTS keywords TEXT[] NOT NULL DEFAULT '{}';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE public.category_modifiers DROP COLUMN IF EXISTS keywords;

-- +goose StatementEnd
