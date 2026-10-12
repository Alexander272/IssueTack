-- Dev-сиды для оборудования. НЕ goose-миграция: применяется вручную на dev-БД,
-- чтобы тестовые данные не уехали на прод.
--
--   psql "postgresql://postgres:postgres@127.0.0.1:5436/issue_track?sslmode=disable" \
--        -f internal/migrate/postgres/seeds/equipment_dev.sql
--
-- Идемпотентно: типы — ON CONFLICT, единицы — по инвентарному номеру.
DO $$
DECLARE
    r_id       UUID;
    t_desktop  UUID;
    t_ram      UUID;
    t_ssd      UUID;
    t_laptop   UUID;
    t_mouse    UUID;
    t_bag      UUID;
    t_printer  UUID;
    e_desktop  UUID;
    e_laptop   UUID;
    u_id       UUID;
BEGIN
    SELECT id INTO r_id FROM realms WHERE code = 'it';
    IF r_id IS NULL THEN
        RAISE NOTICE 'realm "it" not found — equipment seed skipped';
        RETURN;
    END IF;

    INSERT INTO equipment_types (realm_id, name, is_component, sort_order) VALUES
        (r_id, 'Ноутбук', false, 10),
        (r_id, 'Системный блок', false, 20),
        (r_id, 'Монитор', false, 30),
        (r_id, 'Память (RAM)', true, 40),
        (r_id, 'Накопитель (SSD/HDD)', true, 50),
        (r_id, 'Принтер/МФУ', false, 60),
        (r_id, 'Телефон', false, 70),
        (r_id, 'Мышь', true, 80),
        (r_id, 'Сумка', true, 90)
    ON CONFLICT (realm_id, name) DO NOTHING;

    SELECT id INTO t_desktop FROM equipment_types WHERE realm_id = r_id AND name = 'Системный блок';
    SELECT id INTO t_ram     FROM equipment_types WHERE realm_id = r_id AND name = 'Память (RAM)';
    SELECT id INTO t_ssd     FROM equipment_types WHERE realm_id = r_id AND name = 'Накопитель (SSD/HDD)';
    SELECT id INTO t_laptop  FROM equipment_types WHERE realm_id = r_id AND name = 'Ноутбук';
    SELECT id INTO t_mouse   FROM equipment_types WHERE realm_id = r_id AND name = 'Мышь';
    SELECT id INTO t_bag     FROM equipment_types WHERE realm_id = r_id AND name = 'Сумка';
    SELECT id INTO t_printer FROM equipment_types WHERE realm_id = r_id AND name = 'Принтер/МФУ';

    -- Комплект: системный блок.
    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1001') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location, purchase_date, cost)
        VALUES (r_id, t_desktop, 'Системный блок бухгалтерии', 'Dell', 'OptiPlex 3080', 'DL-3080-0001', 'INV-1001', 'in_use', 'Кабинет 305', '2024-02-01', 52000.00)
        RETURNING id INTO e_desktop;
    ELSE
        SELECT id INTO e_desktop FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1001';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1001-01') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, specs, status, location, parent_id)
        VALUES (r_id, t_ram, 'Память Kingston 8GB', 'Kingston', 'KVR26N19S8/8', 'RAM-K-0001', 'INV-1001-01',
                '{"capacityGb": 8, "type": "DDR4"}'::jsonb, 'in_use', 'Кабинет 305', e_desktop);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1001-02') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, specs, status, location, parent_id)
        VALUES (r_id, t_ssd, 'SSD Samsung 512GB', 'Samsung', 'MZ-77E512', 'SSD-S-0001', 'INV-1001-02',
                '{"capacityGb": 512, "type": "SATA"}'::jsonb, 'in_use', 'Кабинет 305', e_desktop);
    END IF;

    -- Комплект: ноутбук + мышь + сумка.
    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location, purchase_date, cost)
        VALUES (r_id, t_laptop, 'Ноутбук менеджера', 'Lenovo', 'ThinkPad T14', 'LN-T14-0002', 'INV-1002', 'in_use', 'Кабинет 401', '2024-06-15', 89000.00)
        RETURNING id INTO e_laptop;
    ELSE
        SELECT id INTO e_laptop FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002-01') THEN
        INSERT INTO equipment (realm_id, type_id, name, serial_number, inventory_number, status, location, parent_id)
        VALUES (r_id, t_mouse, 'Мышь Bluetooth', 'MS-B-0002', 'INV-1002-01', 'in_use', 'Кабинет 401', e_laptop);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002-02') THEN
        INSERT INTO equipment (realm_id, type_id, name, serial_number, inventory_number, status, location, parent_id)
        VALUES (r_id, t_bag, 'Сумка для ноутбука 14"', 'BAG-0002', 'INV-1002-02', 'in_use', 'Кабинет 401', e_laptop);
    END IF;

    -- Разрозненные единицы и склад.
    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1003') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location)
        VALUES (r_id, t_printer, 'Принтер HP LaserJet', 'HP', 'LaserJet M404dn', 'HP-M404-0003', 'INV-1003', 'repair', 'Сервисная зона');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1004') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location)
        VALUES (r_id, t_ram, 'Память Crucial 16GB (резерв)', 'Crucial', 'CT16G4SFRA32A', 'RAM-C-0004', 'INV-1004', 'in_stock', 'Склад, полка A');
    END IF;

    -- Пример истории выдачи: системный блок на активного пользователя реалма.
    SELECT u.id INTO u_id
    FROM users u
    JOIN user_realms ur ON ur.user_id = u.id
    WHERE ur.realm_id = r_id AND ur.is_active = true
    ORDER BY u.created_at
    LIMIT 1;

    IF u_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM equipment_assignments a
        WHERE a.equipment_id = e_desktop AND a.unassigned_at IS NULL
    ) THEN
        INSERT INTO equipment_assignments (equipment_id, user_id, site_id, location, assigned_by, comment)
        VALUES (e_desktop, u_id, NULL, 'Кабинет 305', u_id, 'Плановое закрепление');
    END IF;
END $$;
