-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ticket_notifiers (
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (ticket_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_ticket_notifiers_user_id ON ticket_notifiers(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ticket_notifiers;
-- +goose StatementEnd
