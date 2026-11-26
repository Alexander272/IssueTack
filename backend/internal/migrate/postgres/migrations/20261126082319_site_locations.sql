-- +goose Up
-- +goose StatementBegin
-- Расположения — справочник второго уровня: каждое принадлежит площадке.
-- Площадки глобальные (без realm_id), поэтому справочник тоже глобальный.
CREATE TABLE IF NOT EXISTS site_locations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (site_id, name)
);
CREATE INDEX IF NOT EXISTS idx_site_locations_site ON site_locations (site_id);

CREATE TRIGGER trg_site_locations_updated_at
    BEFORE UPDATE ON site_locations
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE equipment
    ADD COLUMN IF NOT EXISTS location_id uuid REFERENCES site_locations(id) ON DELETE RESTRICT;
ALTER TABLE equipment_assignments
    ADD COLUMN IF NOT EXISTS location_id uuid REFERENCES site_locations(id) ON DELETE RESTRICT;
-- Акт — документ: вместе со ссылкой хранит снимок названия, чтобы переименование
-- расположения не меняло уже сформированный акт.
ALTER TABLE equipment_acts
    ADD COLUMN IF NOT EXISTS location_id uuid REFERENCES site_locations(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS location_name text;

-- Старые CHECK-ы ссылались на текстовую колонку. Имена у них автосгенерированные,
-- поэтому ищем по содержимому определения, а не по фиксированному имени.
DO $mig$
DECLARE
    tbl text;
    con record;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['equipment', 'equipment_assignments'] LOOP
        FOR con IN
            SELECT conname FROM pg_constraint
            WHERE conrelid = tbl::regclass
              AND contype = 'c'
              AND pg_get_constraintdef(oid) ILIKE '%location%'
        LOOP
            EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', tbl, con.conname);
        END LOOP;
    END LOOP;
END
$mig$;

-- Переносить старые текстовые значения не нужно: они относятся к другой модели.
ALTER TABLE equipment DROP COLUMN IF EXISTS location;
ALTER TABLE equipment_assignments DROP COLUMN IF EXISTS location;
ALTER TABLE equipment_acts DROP COLUMN IF EXISTS location;

-- NOT VALID: старые строки с текстовой локацией не бэкфиллятся (это другая
-- модель), поэтому проверять их нельзя — иначе миграция не встанет на БД с
-- существующими данными. Новые и обновлённые строки ограничение соблюдают.
ALTER TABLE equipment ADD CONSTRAINT equipment_place_check
    CHECK (holder_user_id IS NOT NULL OR holder_name IS NOT NULL OR site_id IS NOT NULL OR location_id IS NOT NULL) NOT VALID;
ALTER TABLE equipment_assignments ADD CONSTRAINT equipment_assignments_place_check
    CHECK (user_id IS NOT NULL OR holder_name IS NOT NULL OR site_id IS NOT NULL OR location_id IS NOT NULL) NOT VALID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE equipment DROP CONSTRAINT IF EXISTS equipment_place_check;
ALTER TABLE equipment_assignments DROP CONSTRAINT IF EXISTS equipment_assignments_place_check;

ALTER TABLE equipment ADD COLUMN location text;
ALTER TABLE equipment_assignments ADD COLUMN location text;
ALTER TABLE equipment_acts ADD COLUMN location text;

ALTER TABLE equipment DROP COLUMN IF EXISTS location_id;
ALTER TABLE equipment_assignments DROP COLUMN IF EXISTS location_id;
ALTER TABLE equipment_acts DROP COLUMN IF EXISTS location_id;
ALTER TABLE equipment_acts DROP COLUMN IF EXISTS location_name;

DROP TRIGGER IF EXISTS trg_site_locations_updated_at ON site_locations;
DROP TABLE IF EXISTS site_locations;
-- +goose StatementEnd
