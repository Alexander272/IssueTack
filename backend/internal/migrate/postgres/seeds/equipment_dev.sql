-- Dev-сиды для оборудования. НЕ goose-миграция: применяется вручную на dev-БД,
-- чтобы тестовые данные не уехали на прод.
--
--   psql "postgresql://postgres:postgres@127.0.0.1:5436/issue_track?sslmode=disable" \
--        -f internal/migrate/postgres/seeds/equipment_dev.sql
--
-- Идемпотентно: типы — ON CONFLICT, единицы — по инвентарному номеру.
-- Демонстрирует все категории типов (device/component/periphery/consumable),
-- физический состав (parent_id: комплектующие и расходники) и логический
-- комплект (is_kit + kit_compositions: ноутбук + мышь + сумка).
DO $$
DECLARE
    r_id        UUID;
    t_desktop   UUID;
    t_ram       UUID;
    t_ssd       UUID;
    t_laptop    UUID;
    t_mouse     UUID;
    t_bag       UUID;
    t_printer   UUID;
    t_cartridge UUID;
    e_desktop   UUID;
    e_laptop    UUID;
    e_mouse     UUID;
    e_bag       UUID;
    e_printer   UUID;
    u_id        UUID;
BEGIN
    SELECT id INTO r_id FROM realms WHERE code = 'it';
    IF r_id IS NULL THEN
        RAISE NOTICE 'realm "it" not found — equipment seed skipped';
        RETURN;
    END IF;

    INSERT INTO equipment_types (realm_id, name, kind, sort_order) VALUES
        (r_id, 'Ноутбук', 'device', 10),
        (r_id, 'Системный блок', 'device', 20),
        (r_id, 'Монитор', 'device', 30),
        (r_id, 'Память (RAM)', 'component', 40),
        (r_id, 'Накопитель (SSD/HDD)', 'component', 50),
        (r_id, 'Принтер/МФУ', 'device', 60),
        (r_id, 'Телефон', 'device', 70),
        (r_id, 'Мышь', 'periphery', 80),
        (r_id, 'Сумка', 'periphery', 90),
        (r_id, 'Картридж', 'consumable', 100),
        (r_id, 'Патч-корд (кабель)', 'consumable', 110)
    ON CONFLICT (realm_id, name) DO UPDATE SET kind = EXCLUDED.kind, sort_order = EXCLUDED.sort_order;

    SELECT id INTO t_desktop   FROM equipment_types WHERE realm_id = r_id AND name = 'Системный блок';
    SELECT id INTO t_ram       FROM equipment_types WHERE realm_id = r_id AND name = 'Память (RAM)';
    SELECT id INTO t_ssd       FROM equipment_types WHERE realm_id = r_id AND name = 'Накопитель (SSD/HDD)';
    SELECT id INTO t_laptop    FROM equipment_types WHERE realm_id = r_id AND name = 'Ноутбук';
    SELECT id INTO t_mouse     FROM equipment_types WHERE realm_id = r_id AND name = 'Мышь';
    SELECT id INTO t_bag       FROM equipment_types WHERE realm_id = r_id AND name = 'Сумка';
    SELECT id INTO t_printer   FROM equipment_types WHERE realm_id = r_id AND name = 'Принтер/МФУ';
    SELECT id INTO t_cartridge FROM equipment_types WHERE realm_id = r_id AND name = 'Картридж';

    -- Физический состав: системный блок из комплектующих (parent_id).
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

    -- Логический комплект: ноутбук + мышь + сумка (is_kit + kit_compositions).
    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location, purchase_date, cost, is_kit)
        VALUES (r_id, t_laptop, 'Ноутбук менеджера', 'Lenovo', 'ThinkPad T14', 'LN-T14-0002', 'INV-1002', 'in_use', 'Кабинет 401', '2024-06-15', 89000.00, true)
        RETURNING id INTO e_laptop;
    ELSE
        SELECT id INTO e_laptop FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002-01') THEN
        INSERT INTO equipment (realm_id, type_id, name, serial_number, inventory_number, status, location)
        VALUES (r_id, t_mouse, 'Мышь Bluetooth', 'MS-B-0002', 'INV-1002-01', 'in_use', 'Кабинет 401')
        RETURNING id INTO e_mouse;
    ELSE
        SELECT id INTO e_mouse FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002-01';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002-02') THEN
        INSERT INTO equipment (realm_id, type_id, name, serial_number, inventory_number, status, location)
        VALUES (r_id, t_bag, 'Сумка для ноутбука 14"', 'BAG-0002', 'INV-1002-02', 'in_use', 'Кабинет 401')
        RETURNING id INTO e_bag;
    ELSE
        SELECT id INTO e_bag FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1002-02';
    END IF;

    INSERT INTO kit_compositions (kit_id, component_id, reason)
    SELECT e_laptop, e_mouse, 'сборка комплекта'
    WHERE NOT EXISTS (
        SELECT 1 FROM kit_compositions WHERE kit_id = e_laptop AND component_id = e_mouse AND ended_at IS NULL
    );

    INSERT INTO kit_compositions (kit_id, component_id, reason)
    SELECT e_laptop, e_bag, 'сборка комплекта'
    WHERE NOT EXISTS (
        SELECT 1 FROM kit_compositions WHERE kit_id = e_laptop AND component_id = e_bag AND ended_at IS NULL
    );

    -- Принтер с расходником (картридж) в физическом составе.
    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1003') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location)
        VALUES (r_id, t_printer, 'Принтер HP LaserJet', 'HP', 'LaserJet M404dn', 'HP-M404-0003', 'INV-1003', 'repair', 'Сервисная зона')
        RETURNING id INTO e_printer;
    ELSE
        SELECT id INTO e_printer FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1003';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM equipment WHERE realm_id = r_id AND inventory_number = 'INV-1003-01') THEN
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location, parent_id)
        VALUES (r_id, t_cartridge, 'Картридж HP 58A', 'HP', 'CF258A', 'HP-CF258A-0001', 'INV-1003-01', 'in_use', 'Сервисная зона', e_printer);
    END IF;

    -- Склад.
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
