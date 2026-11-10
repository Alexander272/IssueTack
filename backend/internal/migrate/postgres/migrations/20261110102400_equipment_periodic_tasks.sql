-- +goose Up
-- +goose StatementBegin

-- Шаблоны периодических задач для типов оборудования.
-- Каждая запись определяет: для какого типа, какой заголовок/описание,
-- периодичность и когда создавать следующую задачу.
CREATE TABLE IF NOT EXISTS public.equipment_type_periodic_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id        UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    equipment_type_id UUID NOT NULL REFERENCES equipment_types(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    -- Период между задачами: интервал PostgreSQL (например, '30 days', '3 months')
    period_interval TEXT NOT NULL,
    -- Категория для создаваемых заявок (обязательна)
    category_id     UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    -- Площадка по умолчанию (nullable)
    site_id         UUID REFERENCES sites(id) ON DELETE SET NULL,
    -- Приоритет по умолчанию
    priority        TEXT NOT NULL DEFAULT 'medium',
    -- Создавать ли задачу при первой выдаче оборудования этого типа
    auto_start_on_first_assign BOOLEAN NOT NULL DEFAULT true,
    -- Включено ли правило
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE (realm_id, equipment_type_id, title)
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_etpt_realm_type ON public.equipment_type_periodic_tasks(realm_id, equipment_type_id);
CREATE INDEX IF NOT EXISTS idx_etpt_enabled ON public.equipment_type_periodic_tasks(enabled);

-- Экземпляры периодических задач: связь созданной заявки с шаблоном.
-- Хранит состояние: когда создана, когда следующая, чтобы планировщик знал,
-- когда создавать следующую задачу этого типа для этого оборудования.
CREATE TABLE IF NOT EXISTS public.equipment_periodic_task_instances (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id            UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    template_id         UUID NOT NULL REFERENCES equipment_type_periodic_tasks(id) ON DELETE CASCADE,
    equipment_id        UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    ticket_id           UUID REFERENCES tickets(id) ON DELETE SET NULL,
    -- Статус экземпляра: pending (ожидает создания), created (заявка создана), cancelled (отменён)
    status              TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'created', 'cancelled')),
    -- Когда должна быть создана задача (расчитывается по последней выдаче + период)
    scheduled_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    -- Когда фактически создана (ticket.created_at)
    created_at          TIMESTAMP WITH TIME ZONE,
    -- Когда создавать следующую задачу этого шаблона для этого оборудования
    next_scheduled_at   TIMESTAMP WITH TIME ZONE,
    -- Последняя выдача, по которой рассчитан этот старт (для отладки)
    last_assignment_id  UUID REFERENCES equipment_assignments(id) ON DELETE SET NULL,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,

    UNIQUE (realm_id, template_id, equipment_id, scheduled_at)
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_epti_realm_template ON public.equipment_periodic_task_instances(realm_id, template_id);
CREATE INDEX IF NOT EXISTS idx_epti_equipment ON public.equipment_periodic_task_instances(equipment_id);
CREATE INDEX IF NOT EXISTS idx_epti_status_scheduled
    ON public.equipment_periodic_task_instances(status, scheduled_at)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_epti_ticket ON public.equipment_periodic_task_instances(ticket_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS public.equipment_periodic_task_instances;
DROP TABLE IF EXISTS public.equipment_type_periodic_tasks;

-- +goose StatementEnd
