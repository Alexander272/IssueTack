-- +goose Up
-- +goose StatementBegin

-- Расширяем equipment: логический комплект + признак kit
ALTER TABLE IF EXISTS public.equipment
    ADD COLUMN IF NOT EXISTS is_kit BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS kit_id UUID REFERENCES equipment(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_equipment_kit_id ON public.equipment(kit_id);
CREATE INDEX IF NOT EXISTS idx_equipment_is_kit ON public.equipment(is_kit);

-- История состава комплекта (интервальная). Текущий состав — ended_at IS NULL.
CREATE TABLE IF NOT EXISTS public.kit_compositions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kit_id       UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    component_id UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    started_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at     TIMESTAMP WITH TIME ZONE,
    started_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    ended_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    reason       TEXT NOT NULL DEFAULT '',
    act_id       UUID REFERENCES equipment_acts(id) ON DELETE SET NULL,

    CONSTRAINT kit_compositions_no_self CHECK (kit_id <> component_id),
    CONSTRAINT kit_compositions_open_end CHECK (ended_at IS NULL OR ended_at > started_at)
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_kit_compositions_kit_open
    ON public.kit_compositions(kit_id) WHERE ended_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_kit_compositions_component_open
    ON public.kit_compositions(component_id) WHERE ended_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_kit_compositions_kit
    ON public.kit_compositions(kit_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_kit_compositions_component
    ON public.kit_compositions(component_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_kit_compositions_act_id
    ON public.kit_compositions(act_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_kit_compositions_act_id;
DROP INDEX IF EXISTS idx_kit_compositions_component;
DROP INDEX IF EXISTS idx_kit_compositions_kit;
DROP INDEX IF EXISTS idx_kit_compositions_component_open;
DROP INDEX IF EXISTS idx_kit_compositions_kit_open;

DROP TABLE IF EXISTS public.kit_compositions;

DROP INDEX IF EXISTS idx_equipment_is_kit;
DROP INDEX IF EXISTS idx_equipment_kit_id;

ALTER TABLE IF EXISTS public.equipment
    DROP COLUMN IF EXISTS kit_id,
    DROP COLUMN IF EXISTS is_kit;

-- +goose StatementEnd
