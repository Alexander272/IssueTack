-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.checklist_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    realm_id    UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    title       TEXT COLLATE pg_catalog."default" NOT NULL,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT now()
);

ALTER TABLE IF EXISTS public.checklist_templates
    OWNER to postgres;

CREATE TABLE IF NOT EXISTS public.checklist_template_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
    title       TEXT COLLATE pg_catalog."default" NOT NULL,
    description TEXT COLLATE pg_catalog."default" DEFAULT ''::text,
    sort_order  INT DEFAULT 0
);

ALTER TABLE IF EXISTS public.checklist_template_items
    OWNER to postgres;

CREATE INDEX idx_checklist_items_template ON checklist_template_items(template_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_checklist_items_template;
DROP TABLE IF EXISTS public.checklist_template_items;
DROP TABLE IF EXISTS public.checklist_templates;
-- +goose StatementEnd
