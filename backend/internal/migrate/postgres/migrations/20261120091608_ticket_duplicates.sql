-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS public.tickets
    ADD COLUMN IF NOT EXISTS duplicate_of UUID REFERENCES public.tickets(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS previous_status ticket_status;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'tickets_duplicate_of_check'
    ) THEN
        ALTER TABLE public.tickets
            ADD CONSTRAINT tickets_duplicate_of_check
            CHECK (duplicate_of IS NULL OR (duplicate_of <> id AND previous_status IS NOT NULL));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_tickets_duplicate_of ON public.tickets(duplicate_of);

CREATE TABLE IF NOT EXISTS ticket_duplicate_notifiers (
    primary_ticket_id   UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    duplicate_ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    added_by            UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (primary_ticket_id, user_id, duplicate_ticket_id)
);

CREATE INDEX IF NOT EXISTS idx_ticket_duplicate_notifiers_duplicate
    ON ticket_duplicate_notifiers(duplicate_ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_duplicate_notifiers_user
    ON ticket_duplicate_notifiers(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ticket_duplicate_notifiers;
DROP INDEX IF EXISTS public.idx_tickets_duplicate_of;
ALTER TABLE IF EXISTS public.tickets
    DROP CONSTRAINT IF EXISTS tickets_duplicate_of_check,
    DROP COLUMN IF EXISTS previous_status,
    DROP COLUMN IF EXISTS duplicate_of;
-- +goose StatementEnd
