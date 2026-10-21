-- +goose Up
-- +goose StatementBegin
-- Справочник магазинов реалма: подсказки для поля «Магазин» (autocomplete), чтобы
-- одно и то же название не вводили заново. Единица хранит название текстом
-- (equipment.shop), справочник — только источник вариантов: удаление магазина из
-- справочника не трогает уже заведённые единицы. Новое значение из формы
-- добавляется сюда upsert'ом (см. EquipmentRepo.EnsureShop) в той же транзакции.
CREATE TABLE IF NOT EXISTS equipment_shops (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id   UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (realm_id, name)
);

CREATE INDEX IF NOT EXISTS idx_equipment_shops_realm ON equipment_shops (realm_id, name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS equipment_shops;
-- +goose StatementEnd
