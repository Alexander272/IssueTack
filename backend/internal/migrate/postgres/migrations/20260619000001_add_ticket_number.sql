-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS public.ticket_counters (
    realm_id UUID PRIMARY KEY REFERENCES public.realms(id) ON DELETE CASCADE,
    last_number INT NOT NULL DEFAULT 0
);

ALTER TABLE IF EXISTS public.tickets
    ADD COLUMN IF NOT EXISTS ticket_number INT,
    ADD COLUMN IF NOT EXISTS realm_id UUID REFERENCES public.realms(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tickets_realm_number ON tickets(realm_id, ticket_number);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_tickets_realm_number;

ALTER TABLE IF EXISTS public.tickets
    DROP COLUMN IF EXISTS realm_id,
    DROP COLUMN IF EXISTS ticket_number;

DROP TABLE IF EXISTS public.ticket_counters;

-- +goose StatementEnd
