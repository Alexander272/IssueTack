-- +goose Up
-- +goose StatementBegin

ALTER TABLE public.users ADD COLUMN IF NOT EXISTS is_system boolean DEFAULT false;

INSERT INTO public.realms (code, name, description, is_active)
VALUES ('it', 'IT', 'IT отдел', true)
ON CONFLICT (code) DO NOTHING;

INSERT INTO public.roles (slug, name, realm_id, description, level, is_system, is_editable)
VALUES 
    ('admin', 'Администратор', (SELECT id FROM public.realms WHERE code = 'it'), 'Полный доступ к системе', 100, true, true),
    ('user', 'Пользователь', (SELECT id FROM public.realms WHERE code = 'it'), 'Базовый пользователь', 10, true, true),
    ('chief', 'Начальник', (SELECT id FROM public.realms WHERE code = 'it'), 'Руководитель', 50, true, true),
    ('root', 'Суперпользователь', (SELECT id FROM public.realms WHERE code = 'it'), 'Root-доступ', 1000, true, false)
ON CONFLICT (slug, realm_id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM public.roles WHERE slug IN ('admin', 'user', 'chief', 'root') 
    AND realm_id = (SELECT id FROM public.realms WHERE code = 'it');
DELETE FROM public.realms WHERE code = 'it';
ALTER TABLE public.users DROP COLUMN IF EXISTS is_system;

-- +goose StatementEnd
