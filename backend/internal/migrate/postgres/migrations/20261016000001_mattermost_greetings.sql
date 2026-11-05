-- +goose Up
-- +goose StatementBegin

-- Флаг «пользователю уже отправляли приветствие в Mattermost». Хранит пару
-- realm + пользователь: приветствие приходит один раз на человека в реалме.
-- Строка создаётся атомарно (INSERT ... ON CONFLICT DO NOTHING) до отправки —
-- это защищает от двух одновременных триггеров (direct_added и открытие диалога
-- из плагина). Если отправка не удалась, строка удаляется и попытка повторяется.
CREATE TABLE IF NOT EXISTS public.mattermost_greetings (
    realm_id   UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    greeted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (realm_id, user_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS public.mattermost_greetings;

-- +goose StatementEnd
