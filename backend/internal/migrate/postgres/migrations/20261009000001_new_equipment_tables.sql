-- +goose Up
-- +goose StatementBegin

-- Статусы единицы оборудования.
CREATE TYPE equipment_status AS ENUM (
    'in_stock',
    'in_use',
    'repair',
    'reserved',
    'written_off'
);

-- Типы событий журнала оборудования. operation_id в equipment_events группирует
-- события одной операции (например, component_removed + component_added при замене).
CREATE TYPE equipment_event_type AS ENUM (
    'created',
    'updated',
    'assigned',
    'unassigned',
    'moved',
    'status_changed',
    'component_added',
    'component_removed',
    'component_replaced',
    'repaired',
    'written_off'
);

-- Справочник типов оборудования (настраиваемый на реалм).
CREATE TABLE IF NOT EXISTS public.equipment_types (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id     UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    is_component BOOLEAN NOT NULL DEFAULT false,
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE (realm_id, name)
)
TABLESPACE pg_default;

-- Единица оборудования. parent_id — дерево комплектов (системник → RAM/SSD,
-- комплект → ноут+мышь+сумка). Текущий держатель/локация денормализованы для
-- списков; история — в equipment_assignments. Каждая единица обязана быть
-- физически учтена: держатель (пользователь или ФИО) или локация (площадка
-- или текст) — хотя бы одно, иначе «где это лежит» неизвестно.
CREATE TABLE IF NOT EXISTS public.equipment (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id         UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    type_id          UUID NOT NULL REFERENCES equipment_types(id),
    name             TEXT NOT NULL,
    vendor           TEXT NOT NULL DEFAULT '',
    model            TEXT NOT NULL DEFAULT '',
    serial_number    TEXT,
    inventory_number TEXT,
    specs            JSONB NOT NULL DEFAULT '{}'::jsonb,
    status           equipment_status NOT NULL DEFAULT 'in_stock',
    parent_id        UUID REFERENCES equipment(id) ON DELETE SET NULL,
    site_id          UUID REFERENCES sites(id) ON DELETE SET NULL,
    holder_user_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    holder_name      TEXT,
    location         TEXT,
    purchase_date    DATE,
    warranty_until   DATE,
    cost             NUMERIC(12, 2),
    notes            TEXT NOT NULL DEFAULT '',
    created_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CHECK (holder_user_id IS NOT NULL OR holder_name IS NOT NULL OR site_id IS NOT NULL OR location IS NOT NULL)
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_equipment_parent_id ON public.equipment(parent_id);
CREATE INDEX IF NOT EXISTS idx_equipment_holder_user_id ON public.equipment(holder_user_id);
CREATE INDEX IF NOT EXISTS idx_equipment_realm_status ON public.equipment(realm_id, status);
CREATE INDEX IF NOT EXISTS idx_equipment_realm_type ON public.equipment(realm_id, type_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_equipment_realm_serial
    ON public.equipment(realm_id, serial_number)
    WHERE serial_number IS NOT NULL AND serial_number <> '';
CREATE UNIQUE INDEX IF NOT EXISTS uq_equipment_realm_inventory
    ON public.equipment(realm_id, inventory_number)
    WHERE inventory_number IS NOT NULL AND inventory_number <> '';

-- Автообновление updated_at (страховка: у оборудования много UPDATE-путей —
-- смена статуса, пропагация выдач, состав), хотя репозитории и так пишут
-- updated_at=NOW() явно, как и во всём проекте.
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_equipment_updated_at
    BEFORE UPDATE ON public.equipment
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_equipment_types_updated_at
    BEFORE UPDATE ON public.equipment_types
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- История выдач/локаций. Открытая строка (unassigned_at IS NULL) — текущая.
-- Держатель задаётся пользователем либо ФИО текстом (внешние/уволенные), а также
-- площадкой либо уточнением локации — хотя бы одним из полей.
CREATE TABLE IF NOT EXISTS public.equipment_assignments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    equipment_id  UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    holder_name   TEXT,
    site_id       UUID REFERENCES sites(id) ON DELETE SET NULL,
    location      TEXT,
    assigned_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    unassigned_at TIMESTAMP WITH TIME ZONE,
    assigned_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    comment       TEXT NOT NULL DEFAULT '',

    CHECK (user_id IS NOT NULL OR holder_name IS NOT NULL OR site_id IS NOT NULL OR location IS NOT NULL)
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_equipment_assignments_equipment
    ON public.equipment_assignments(equipment_id, unassigned_at);
CREATE INDEX IF NOT EXISTS idx_equipment_assignments_user
    ON public.equipment_assignments(user_id);

-- Иммутабельный журнал событий: читаемый таймлайн + группировка операций.
CREATE TABLE IF NOT EXISTS public.equipment_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    equipment_id  UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    event_type    equipment_event_type NOT NULL,
    operation_id  UUID NOT NULL,
    component_id  UUID REFERENCES equipment(id) ON DELETE SET NULL,
    old_value     JSONB,
    new_value     JSONB,
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    comment       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_equipment_events_equipment
    ON public.equipment_events(equipment_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_equipment_events_operation ON public.equipment_events(operation_id);
CREATE INDEX IF NOT EXISTS idx_equipment_events_component ON public.equipment_events(component_id);

-- Акты выдачи со снимком состава комплекта на момент выдачи.
CREATE TABLE IF NOT EXISTS public.equipment_acts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id          UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    number            TEXT NOT NULL,
    kit_equipment_id  UUID REFERENCES equipment(id) ON DELETE SET NULL,
    recipient_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    recipient_name    TEXT,
    site_id           UUID REFERENCES sites(id) ON DELETE SET NULL,
    location          TEXT,
    issued_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    issued_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    content           JSONB NOT NULL DEFAULT '[]'::jsonb,
    comment           TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE (realm_id, number)
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_equipment_acts_realm ON public.equipment_acts(realm_id, issued_at DESC);

-- Связь заявки с оборудованием (UI — отдельным этапом; колонка уже здесь,
-- чтобы не делать миграцию в двух местах).
ALTER TABLE IF EXISTS public.tickets
    ADD COLUMN IF NOT EXISTS equipment_id UUID REFERENCES equipment(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tickets_equipment_id ON public.tickets(equipment_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_tickets_equipment_id;
ALTER TABLE IF EXISTS public.tickets DROP COLUMN IF EXISTS equipment_id;

DROP TRIGGER IF EXISTS trg_equipment_updated_at ON public.equipment;
DROP TRIGGER IF EXISTS trg_equipment_types_updated_at ON public.equipment_types;
DROP FUNCTION IF EXISTS public.set_updated_at();

DROP TABLE IF EXISTS public.equipment_acts;
DROP TABLE IF EXISTS public.equipment_events;
DROP TABLE IF EXISTS public.equipment_assignments;
DROP TABLE IF EXISTS public.equipment;
DROP TABLE IF EXISTS public.equipment_types;

DROP TYPE IF EXISTS equipment_event_type;
DROP TYPE IF EXISTS equipment_status;

-- +goose StatementEnd
