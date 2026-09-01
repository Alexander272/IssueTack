-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS public.ticket_favorites (
    id         UUID NOT NULL,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ticket_id  UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    type       TEXT NOT NULL DEFAULT 'permanent',
    created_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (id),
    CONSTRAINT uq_ticket_favorites_user_ticket_type UNIQUE (user_id, ticket_id, type)
);

CREATE INDEX IF NOT EXISTS idx_ticket_favorites_user ON ticket_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_ticket_favorites_ticket ON ticket_favorites(ticket_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_ticket_favorites_ticket;
DROP INDEX IF EXISTS idx_ticket_favorites_user;

DROP TABLE IF EXISTS public.ticket_favorites;

-- +goose StatementEnd
