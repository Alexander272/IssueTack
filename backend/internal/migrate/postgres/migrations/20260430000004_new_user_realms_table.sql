-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.user_realms
(
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    realm_id uuid NOT NULL,
    role_id uuid NOT NULL,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT user_realms_pkey PRIMARY KEY (id),
    CONSTRAINT user_realms_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT user_realms_realm_id_fkey FOREIGN KEY (realm_id)
        REFERENCES public.realms (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT user_realms_role_id_fkey FOREIGN KEY (role_id)
        REFERENCES public.roles (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT user_realms_unique UNIQUE (user_id, realm_id)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.user_realms
    OWNER to postgres;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.user_realms;
-- +goose StatementEnd
