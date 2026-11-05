-- +goose Up
-- +goose StatementBegin

-- Скрытые категории и разделы категорий: их не видят «обычные пользователи»
-- (безгрупповые заказчики), а менеджеры и исполнители видят. Скрытость раздела
-- делает скрытыми все его категории — эффективно, без записи в строки: фильтр
-- в запросах учитывает `categories.is_hidden` ИЛИ `category_groups.is_hidden`.
ALTER TABLE IF EXISTS public.categories
    ADD COLUMN IF NOT EXISTS is_hidden BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE IF EXISTS public.category_groups
    ADD COLUMN IF NOT EXISTS is_hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.category_groups DROP COLUMN IF EXISTS is_hidden;
ALTER TABLE IF EXISTS public.categories DROP COLUMN IF EXISTS is_hidden;

-- +goose StatementEnd
