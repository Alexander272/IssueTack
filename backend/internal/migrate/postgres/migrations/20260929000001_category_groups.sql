-- +goose Up
-- +goose StatementBegin

-- Разделы категорий — таксономия поверх групп-владельцев.
-- categories.group_id остаётся группой-владельцем (маршрутизация, менеджер),
-- а category_group_id только группирует категории в списках выбора.
CREATE TABLE IF NOT EXISTS public.category_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id    UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name        TEXT COLLATE pg_catalog."default" NOT NULL,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(realm_id, name)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.category_groups
    OWNER to postgres;

ALTER TABLE IF EXISTS public.categories
    ADD COLUMN IF NOT EXISTS category_group_id UUID REFERENCES category_groups(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_categories_category_group_id ON public.categories(category_group_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_categories_category_group_id;
ALTER TABLE IF EXISTS public.categories DROP COLUMN IF EXISTS category_group_id;
DROP TABLE IF EXISTS public.category_groups;

-- +goose StatementEnd
