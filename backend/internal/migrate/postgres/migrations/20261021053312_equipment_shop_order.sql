-- +goose Up
-- +goose StatementBegin
-- Магазин и номер заказа — реквизиты закупки, отдельные колонки (как vendor/model),
-- а не ключи в specs: по ним нужно искать/сортировать, и они есть у каждой единицы,
-- тогда как specs — свободные характеристики конкретного типа.
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS shop TEXT;
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS order_number TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE equipment DROP COLUMN IF EXISTS shop;
ALTER TABLE equipment DROP COLUMN IF EXISTS order_number;
-- +goose StatementEnd
