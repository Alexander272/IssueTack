-- +goose Up
-- +goose StatementBegin

-- У комплекта нет типа: type_id обязателен для единицы, но не для комплекта.
-- Вставка комплекта пишет NULL, чтение — LEFT JOIN к справочнику типов, поэтому
-- NOT NULL здесь больше не нужен. Единица без типа отсекается в сервисе
-- (EquipmentService.Create/Update), а не ограничением БД: одно поле обслуживает
-- и единицы, и комплекты, и разделить их можно только в коде.
ALTER TABLE public.equipment
    ALTER COLUMN type_id DROP NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Откат вернёт NOT NULL, но выполнится только если комплектов без типа уже нет:
-- заполнить type_id осмысленным значением нельзя, поэтому миграция падает на
-- существующих NULL, а не придумывает тип. Перед откатом комплекты без типа
-- нужно удалить или проставить им любой тип вручную.
ALTER TABLE public.equipment
    ALTER COLUMN type_id SET NOT NULL;

-- +goose StatementEnd
