-- +goose Up
-- +goose StatementBegin
DO $$ 
BEGIN 
    CREATE TYPE ticket_status AS ENUM ('open', 'in_progress', 'pending', 'on_hold', 'resolved', 'closed', 'cancelled');
EXCEPTION 
    WHEN duplicate_object THEN NULL; 
END $$;

CREATE TABLE IF NOT EXISTS public.tickets
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT COLLATE pg_catalog."default" NOT NULL,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    status      ticket_status NOT NULL DEFAULT 'open',
    priority    ticket_priority NOT NULL DEFAULT 'medium',
    site_id     UUID REFERENCES sites(id) ON DELETE SET NULL,
    creator_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id    UUID REFERENCES groups(id) ON DELETE SET NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,        
    assignee_id UUID,        
    manager_id  UUID,        
    due_date    TIMESTAMP WITH TIME ZONE,
    closed_at   TIMESTAMP WITH TIME ZONE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT now()
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.tickets
    OWNER to postgres;

CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_assignee ON tickets(assignee_id);
CREATE INDEX IF NOT EXISTS idx_tickets_owner ON tickets(owner_id);
CREATE INDEX IF NOT EXISTS idx_tickets_group_id ON tickets(group_id);
CREATE INDEX IF NOT EXISTS idx_tickets_category_id ON tickets(category_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tickets_status;
DROP INDEX IF EXISTS idx_tickets_assignee;
DROP INDEX IF EXISTS idx_tickets_owner;
DROP INDEX IF EXISTS idx_tickets_group_id;
DROP INDEX IF EXISTS idx_tickets_category_id;

DROP TABLE IF EXISTS public.tickets;

DROP TYPE IF EXISTS ticket_status;
-- +goose StatementEnd
