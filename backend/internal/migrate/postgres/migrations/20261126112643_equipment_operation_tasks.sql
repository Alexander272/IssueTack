-- +goose Up
-- +goose StatementBegin

-- Правила автоматических задач оборудования: что делать после операции с единицей.
-- Сейчас одно действие — создать заявку; operation/action выбираются из CHECK'ов,
-- чтобы позже добавлять новые правила (новые операции/действия) без смены модели
-- пользовательского интерфейса.
CREATE TABLE IF NOT EXISTS public.equipment_operation_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id    UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    operation   TEXT NOT NULL CHECK (operation IN ('assign', 'unassign')),
    action      TEXT NOT NULL CHECK (action IN ('create_ticket')),
    -- Заголовок/описание заявки. Пусто — берётся стандартный текст 1C:ERP.
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    group_id    UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    priority    TEXT NOT NULL DEFAULT 'medium',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE (realm_id, operation, name)
)
TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_eot_realm_operation
    ON public.equipment_operation_tasks(realm_id, operation) WHERE enabled;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS public.equipment_operation_tasks;

-- +goose StatementEnd
