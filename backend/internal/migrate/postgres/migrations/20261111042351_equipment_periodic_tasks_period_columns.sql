-- +goose Up
-- +goose StatementBegin

-- Период задачи теперь задаётся тремя целыми полями (годы/месяцы/дни) вместо
-- интервала в свободном формате PostgreSQL ('30 days', '3 months'). Компоненты
-- периода применяются последовательно через date arithmetic: сначала годы, затем
-- месяцы, затем дни (Go AddDate — тот же порядок).
ALTER TABLE public.equipment_type_periodic_tasks
    DROP COLUMN IF EXISTS period_interval;

ALTER TABLE public.equipment_type_periodic_tasks
    ADD COLUMN period_years INT NOT NULL DEFAULT 0,
    ADD COLUMN period_months INT NOT NULL DEFAULT 0,
    ADD COLUMN period_days INT NOT NULL DEFAULT 0;

-- Автор шаблона: кто создал правило периодической задачи. Заявки, порождаемые
-- шаблоном, создаются от имени автора (актор = создатель заявки). NULL — шаблон,
-- созданный до миграции (старый формат без автора): такие задачи не стартуют
-- автоматически и требуют ручного запуска (см. ManualPeriodicTaskRunDTO).
ALTER TABLE public.equipment_type_periodic_tasks
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE public.equipment_type_periodic_tasks
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE public.equipment_type_periodic_tasks
    DROP COLUMN IF EXISTS period_days,
    DROP COLUMN IF EXISTS period_months,
    ADD COLUMN period_interval TEXT NOT NULL DEFAULT '30 days';

-- +goose StatementEnd
