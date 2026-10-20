-- +goose Up
-- +goose StatementBegin

-- Ожидаемый срок возврата выданной единицы. Хранится в строке выдачи, а не в
-- самой единице: срок — реквизит конкретной выдачи (у следующей выдачи он свой),
-- а закрытая строка становится историей с сохранённым сроком.
--
-- Пусто (NULL) = срок не задавали: выдают и бессрочно. По этой дате cron шлёт
-- напоминание тому, кто выдал (assigned_by).
ALTER TABLE public.equipment_assignments
    ADD COLUMN IF NOT EXISTS return_due_at TIMESTAMP WITH TIME ZONE;

-- Напоминание ищет открытые выдачи со сроком в прошлом — частичный индекс ровно
-- под этот запрос (закрытые строки и строки без срока не нужны).
CREATE INDEX IF NOT EXISTS idx_equipment_assignments_return_due
    ON public.equipment_assignments(return_due_at)
    WHERE unassigned_at IS NULL AND return_due_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS public.idx_equipment_assignments_return_due;
ALTER TABLE public.equipment_assignments
    DROP COLUMN IF EXISTS return_due_at;

-- +goose StatementEnd
