-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.subtasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    title       TEXT COLLATE pg_catalog."default" NOT NULL,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    status      ticket_status NOT NULL DEFAULT 'open',
    priority    ticket_priority NOT NULL DEFAULT 'medium',
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date    TIMESTAMP WITH TIME ZONE,
    closed_at   TIMESTAMP WITH TIME ZONE,
    sort_order  INT DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT now()
);

ALTER TABLE IF EXISTS public.subtasks
    OWNER to postgres;

CREATE INDEX idx_subtasks_ticket ON subtasks(ticket_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_subtasks_ticket;
DROP TABLE IF EXISTS public.subtasks;
-- +goose StatementEnd
