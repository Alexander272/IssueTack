-- +goose Up
-- +goose StatementBegin

-- Тип события «зафиксирован дефект»: несёт описание неисправности и дату
-- обнаружения (в new_value). Отдельный тип, а не status_changed, потому что
-- дефект — это факт с данными, а не просто смена состояния: лента истории
-- печатает по нему описание, а не «статус X → Y».
-- Значение дописывается в enum (PG 12+ разрешает ADD VALUE в транзакции, если
-- новое значение не используется в этой же транзакции).
ALTER TYPE equipment_event_type ADD VALUE IF NOT EXISTS 'defective';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Удалить значение enum на месте нельзя — пересоздаём тип без 'defective'.
-- События с этим типом уводим в 'status_changed', чтобы не потерять записи
-- журнала (дефект всё равно менял статус, так что переход корректен).
ALTER TYPE equipment_event_type RENAME TO equipment_event_type_old;
CREATE TYPE equipment_event_type AS ENUM (
    'created', 'updated', 'assigned', 'unassigned', 'moved', 'status_changed',
    'component_added', 'component_removed', 'component_replaced', 'repaired', 'written_off'
);
ALTER TABLE IF EXISTS public.equipment_events
    ALTER COLUMN event_type TYPE equipment_event_type
    USING (CASE WHEN event_type::text = 'defective' THEN 'status_changed' ELSE event_type::text END)::equipment_event_type;
DROP TYPE IF EXISTS equipment_event_type_old;

-- +goose StatementEnd
