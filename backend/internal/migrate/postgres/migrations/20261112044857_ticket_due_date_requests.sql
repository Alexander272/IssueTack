-- +goose Up
CREATE TABLE IF NOT EXISTS ticket_due_date_requests (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ticket_id          UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  requester_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  current_due_date   TIMESTAMPTZ,
  proposed_due_date TIMESTAMPTZ NOT NULL,
  reason             TEXT NOT NULL DEFAULT '',
  status             TEXT NOT NULL DEFAULT 'pending',
  decided_by         UUID REFERENCES users(id) ON DELETE SET NULL,
  decided_at         TIMESTAMPTZ,
  decision_comment  TEXT NOT NULL DEFAULT '',
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ticket_due_date_requests_ticket ON ticket_due_date_requests(ticket_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_due_date_requests_one_pending
  ON ticket_due_date_requests(ticket_id)
  WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_ticket_due_date_requests_one_pending;
DROP INDEX IF EXISTS idx_ticket_due_date_requests_ticket;
DROP TABLE IF EXISTS ticket_due_date_requests;
