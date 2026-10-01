-- +goose Up
-- +goose StatementBegin

-- Происхождение пользователя. Канонический источник — Keycloak ('keycloak'),
-- оттуда приходят все сотрудники. Пользователи, созданные из Mattermost
-- (uuid.New(), без учётной записи в Keycloak), помечаются 'mattermost'.
--
-- Колонка нужна синхронизации: userService.Sync удаляет всех, кого нет в
-- группе Keycloak, а tickets.creator_id объявлен как ON DELETE CASCADE —
-- без этого флага синк сносил бы заявки заявителей вместе с комментариями
-- и вложениями. Поэтому же CHECK: опечатка в значении молча превратила бы
-- синк в массовое удаление заявок.
--
-- ВАЖНО, РУЧНОЙ ШАГ ДО ПЕРВОГО ЗАПУСКА Sync: DEFAULT помечает уже существующие
-- строки как 'keycloak', а значит пользователи, созданные ботом ДО этой миграции,
-- останутся удаляемыми — синк снесёт их вместе с заявками. В БД нет признака,
-- отличающего такого пользователя от сотрудника Keycloak (id обоих — UUID), поэтому
-- переопределять их наугад нельзя: ложное 'mattermost' навсегда исключит сотрудника
-- из удаления, ложное 'keycloak' — унесёт заявки. Список таких пользователей нужно
-- получить из Keycloak (их нет в группе синхронизации, но id = sub) и подтвердить
-- вручную, например:
--
--   SELECT id, username, email, mattermost_id, created_at
--   FROM public.users
--   WHERE mattermost_id <> '' AND is_system = false
--   ORDER BY created_at;
--
--   UPDATE public.users SET source = 'mattermost' WHERE id = '<uuid>';
--
-- Пока бэкфилл не сделан, Sync запускать нельзя.
ALTER TABLE IF EXISTS public.users
    ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'keycloak';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_source_check'
    ) THEN
        ALTER TABLE public.users
            ADD CONSTRAINT users_source_check CHECK (source IN ('keycloak', 'mattermost'));
    END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS idx_users_source ON public.users(source);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_users_source;

ALTER TABLE IF EXISTS public.users
    DROP CONSTRAINT IF EXISTS users_source_check;

ALTER TABLE IF EXISTS public.users
    DROP COLUMN IF EXISTS source;

-- +goose StatementEnd
