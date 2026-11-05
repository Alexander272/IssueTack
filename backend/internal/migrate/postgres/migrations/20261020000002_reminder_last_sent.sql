-- +goose Up
CREATE TABLE IF NOT EXISTS reminder_last_sent (
  ticket_id UUID NOT NULL,
  event     TEXT NOT NULL,
  sent_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (ticket_id, event)
);

CREATE INDEX IF NOT EXISTS idx_reminder_last_sent_sent_at ON reminder_last_sent (sent_at);

-- +goose Down
DROP INDEX IF EXISTS idx_reminder_last_sent_sent_at;
DROP TABLE IF EXISTS reminder_last_sent;