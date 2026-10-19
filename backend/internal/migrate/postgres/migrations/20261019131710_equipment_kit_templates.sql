-- +goose Up
-- +goose StatementBegin

-- Шаблоны комплектов: именованный набор типов оборудования, из которого
-- собирают логический комплект («Стандартное рабочее место»). Позиция шаблона —
-- именно ТИП, а не конкретная единица: при создании комплекта по шаблону слоты
-- заполняются типами, а единицы выбираются со склада вручную.
CREATE TABLE IF NOT EXISTS public.equipment_kit_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id    UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE (realm_id, name)
)
TABLESPACE pg_default;

-- Позиции шаблона. type_id с ON DELETE CASCADE: без типа позиция бессмысленна,
-- а удалению типа, не занятого единицами, ничто не мешает.
CREATE TABLE IF NOT EXISTS public.equipment_kit_template_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES equipment_kit_templates(id) ON DELETE CASCADE,
    type_id     UUID NOT NULL REFERENCES equipment_types(id) ON DELETE CASCADE,
    sort_order  INTEGER NOT NULL DEFAULT 0
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_equipment_kit_template_items_template
    ON public.equipment_kit_template_items(template_id);

-- updated_at шаблона ведёт общий триггер (см. new_equipment_tables).
CREATE TRIGGER trg_equipment_kit_templates_updated_at
    BEFORE UPDATE ON public.equipment_kit_templates
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trg_equipment_kit_templates_updated_at ON public.equipment_kit_templates;
DROP TABLE IF EXISTS public.equipment_kit_template_items;
DROP TABLE IF EXISTS public.equipment_kit_templates;

-- +goose StatementEnd
