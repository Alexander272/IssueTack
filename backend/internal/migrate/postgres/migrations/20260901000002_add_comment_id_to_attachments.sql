-- +goose Up
-- +goose StatementBegin

ALTER TABLE public.attachments
    ADD COLUMN comment_id UUID REFERENCES comments(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_attachments_comment ON attachments(comment_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_attachments_comment;

ALTER TABLE public.attachments
    DROP COLUMN IF EXISTS comment_id;

-- +goose StatementEnd