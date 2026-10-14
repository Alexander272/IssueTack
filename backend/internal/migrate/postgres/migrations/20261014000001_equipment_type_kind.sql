-- +goose Up
-- +goose StatementBegin

-- Категория типа оборудования: задаёт группу в справочнике и допустимость быть
-- частью физического состава (parent_id). Заменяет булев is_component, который
-- смешивал «классификацию» и «можно ли вложить».
CREATE TYPE equipment_type_kind AS ENUM (
    'device',
    'component',
    'periphery',
    'consumable'
);

ALTER TABLE IF EXISTS public.equipment_types
    ADD COLUMN IF NOT EXISTS kind equipment_type_kind NOT NULL DEFAULT 'device';

-- Бэкфилл из прежнего флага и эвристики по имени, чтобы текущие данные не
-- потеряли группу: мышь/сумка → периферия, картридж/кабель → расходник,
-- остальные комплектующие → комплектующие, прочее → устройства.
UPDATE public.equipment_types SET kind = CASE
    WHEN name ILIKE '%мыш%' OR name ILIKE '%клав%' OR name ILIKE '%сумк%'
        OR name ILIKE '%перифер%' OR name ILIKE '%наушник%' OR name ILIKE '%гарнитур%'
        OR name ILIKE '%колонк%'
        THEN 'periphery'::equipment_type_kind
    WHEN name ILIKE '%картридж%' OR name ILIKE '%кабел%' OR name ILIKE '%тонер%'
        OR name ILIKE '%бумаг%' OR name ILIKE '%батаре%' OR name ILIKE '%аккумул%'
        THEN 'consumable'::equipment_type_kind
    WHEN is_component THEN 'component'::equipment_type_kind
    ELSE 'device'::equipment_type_kind
END;

ALTER TABLE IF EXISTS public.equipment_types DROP COLUMN IF EXISTS is_component;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS public.equipment_types
    ADD COLUMN IF NOT EXISTS is_component BOOLEAN NOT NULL DEFAULT false;

UPDATE public.equipment_types
    SET is_component = (kind IN ('component', 'consumable'));

ALTER TABLE IF EXISTS public.equipment_types DROP COLUMN IF EXISTS kind;

DROP TYPE IF EXISTS equipment_type_kind;

-- +goose StatementEnd
