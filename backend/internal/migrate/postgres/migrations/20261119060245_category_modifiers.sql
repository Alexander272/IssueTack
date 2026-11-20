-- +goose Up
-- +goose StatementBegin

-- Подкатегории (модификаторы категорий): справочник параметров заявки,
-- привязанный к категории. NULL в default_due_days/priority означает
-- наследование соответствующего значения категории.
CREATE TABLE IF NOT EXISTS public.category_modifiers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id          UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    category_id       UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    default_due_days  INT NULL,
    priority          ticket_priority NULL,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order        INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(realm_id, category_id, name)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.category_modifiers OWNER to postgres;

CREATE INDEX IF NOT EXISTS idx_category_modifiers_category_id ON public.category_modifiers(category_id);

-- Одна подкатегория на заявку. Удаление справочника не должно ронять заявки:
-- историческое значение остаётся в журнале, а поле просто очищается.
ALTER TABLE public.tickets
    ADD COLUMN IF NOT EXISTS category_modifier_id UUID REFERENCES category_modifiers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tickets_category_modifier_id ON public.tickets(category_modifier_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_tickets_category_modifier_id;
ALTER TABLE public.tickets DROP COLUMN IF EXISTS category_modifier_id;
DROP INDEX IF EXISTS idx_category_modifiers_category_id;
DROP TABLE IF EXISTS public.category_modifiers;

-- +goose StatementEnd
