-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS public.ticket_subscriptions (
    ticket_id  UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (ticket_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_ticket_subscriptions_user ON ticket_subscriptions(user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_ticket_subscriptions_user;

DROP TABLE IF EXISTS public.ticket_subscriptions;

-- +goose StatementEnd
