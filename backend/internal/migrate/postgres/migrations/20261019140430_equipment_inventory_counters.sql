-- +goose Up
-- +goose StatementBegin

-- Автоинвентарные номера: счётчик на реалм и серию. Единицы нумеруются только
-- цифрами, комплекты — отдельной серией с префиксом «KIT-», поэтому счётчиков
-- на реалм два (series = 'unit' | 'kit'). Отдельная таблица, а не SEQUENCE:
-- одна последовательность на все реалмы давала бы дыры в нумерации каждого,
-- а своя последовательность на реалм — это DDL при создании реалма. Увеличение
-- идёт в той же транзакции, что и вставка единицы, поэтому откат не «сжигает»
-- номер.
CREATE TABLE IF NOT EXISTS public.equipment_inventory_counters (
    realm_id   UUID NOT NULL REFERENCES public.realms(id) ON DELETE CASCADE,
    series     TEXT NOT NULL CHECK (series IN ('unit', 'kit')),
    last_value BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (realm_id, series)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS public.equipment_inventory_counters;

-- +goose StatementEnd
