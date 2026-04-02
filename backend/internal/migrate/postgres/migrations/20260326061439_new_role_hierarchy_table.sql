-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.role_hierarchy (
    parent_role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    role_id  uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,

    PRIMARY KEY (parent_role_id, role_id),

    CHECK (parent_role_id <> role_id)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.role_hierarchy
    OWNER to postgres;

-- === ЗАЩИТА ОТ ЦИКЛИЧЕСКОГО НАСЛЕДОВАНИЯ (триггер) ===
CREATE OR REPLACE FUNCTION check_role_hierarchy_cycle()
RETURNS TRIGGER AS $$
DECLARE
    cycle_exists BOOLEAN;
BEGIN
    -- Рекурсивная проверка: не ведёт ли новая связь к циклу?
    WITH RECURSIVE role_chain AS (
        SELECT parent_role_id 
        FROM role_hierarchy 
        WHERE role_id = NEW.role_id
        UNION
        SELECT ri.parent_role_id 
        FROM role_hierarchy ri
        INNER JOIN role_chain rc ON ri.role_id = rc.parent_role_id 
        WHERE ri.domain = NEW.domain
    )
    SELECT EXISTS (
        SELECT 1 FROM role_chain WHERE parent_role_id = NEW.parent_role_id
    ) INTO cycle_exists;
    
    IF cycle_exists THEN
        RAISE EXCEPTION 'ERR_CIRCULAR: Circular inheritance detected: role % cannot inherit from %', 
            NEW.role_id, NEW.parent_role_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER IF NOT EXISTS trg_role_hierarchy_cycle
    BEFORE INSERT OR UPDATE ON role_hierarchy
    FOR EACH ROW EXECUTE FUNCTION check_role_hierarchy_cycle();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_role_hierarchy_cycle ON role_hierarchy;
DROP FUNCTION IF EXISTS check_role_hierarchy_cycle;

DROP TABLE IF EXISTS public.role_hierarchy;
-- +goose StatementEnd
