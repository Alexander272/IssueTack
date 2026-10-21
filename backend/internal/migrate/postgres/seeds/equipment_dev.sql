-- Dev-сиды для оборудования. НЕ goose-миграция: применяется вручную на dev-БД,
-- чтобы тестовые данные не уехали на прод.
--
--   psql "postgresql://postgres:postgres@127.0.0.1:5436/issue_track?sslmode=disable" \
--        -f internal/migrate/postgres/seeds/equipment_dev.sql
--
-- Идемпотентно: типы — upsert по имени, единицы — insert-if-absent по инвентарному
-- номеру. Кроме вставки отсутствующих строк сид **приводит форму демо-данных к
-- текущей модели** (блок «приведение к текущей модели»): на БД, где лежат данные
-- прежних версий сида, он их починит, а не молча пропустит. Это важно, потому что
-- до появления `kind` комплект хранился в физсоставе — мышь и сумка были вложены
-- в ноутбук через `parent_id`, а `is_kit` не выставлялся.
--
-- Демонстрирует распространённый офисный набор типов во всех категориях
-- (device/component/periphery/consumable), физический состав (parent_id:
-- комплектующие в системном блоке, расходник в принтере), логический комплект
-- (is_kit + kit_compositions: ноутбук + мышь + сумка) и выдачу единицы с
-- пропагацией на её состав.
--
-- Журнал (`equipment_events`) и акты (`equipment_acts`) не эмулируются: это
-- append-only история, которую пишут сервисы. Сид заполняет то, что читается
-- напрямую (дерево, состав, держатель, статус), и повторяет денормализацию
-- сервисов — `holder_*`/`status` на поддереве (как `Assign`) и `kit_id` у члена
-- комплекта (как `OpenComposition`). Без неё список и карточка показывают не то,
-- что записано в БД («На складе» при открытой выдаче, «Держатель» у члена
-- комплекта).
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
    v_eq        UUID;
BEGIN
    SELECT id INTO r_id FROM realms WHERE code = 'it';
    IF r_id IS NULL THEN
        RAISE NOTICE 'realm "it" not found — equipment seed skipped';
        RETURN;
    END IF;

    -- Действующее лицо демо-операций: тот, кто уже записан держателем в открытой
    -- выдаче (повторный прогон — держим связь выдачи и денормализации), иначе
    -- первый активный пользователь реалма. Тай-брейк по username — у сид-пользователей
    -- одинаковый created_at, без него выбор недетерминирован.
    SELECT COALESCE(
        (
            SELECT a.user_id
            FROM equipment_assignments a
            JOIN equipment e ON e.id = a.equipment_id
            WHERE e.realm_id = r_id
              AND e.inventory_number = 'INV-1001'
              AND a.unassigned_at IS NULL
              AND a.user_id IS NOT NULL
            ORDER BY a.assigned_at DESC
            LIMIT 1
        ),
        (
            SELECT u.id
            FROM users u
            JOIN user_realms ur ON ur.user_id = u.id
            WHERE ur.realm_id = r_id AND ur.is_active = true
            ORDER BY u.created_at, u.username
            LIMIT 1
        )
    ) INTO u_id;

    INSERT INTO equipment_types (realm_id, name, kind, sort_order) VALUES
        (r_id, 'Ноутбук', 'device', 10),
        (r_id, 'Системный блок', 'device', 20),
        (r_id, 'Монитор', 'device', 30),
        (r_id, 'Принтер/МФУ', 'device', 40),
        (r_id, 'Сканер', 'device', 50),
        (r_id, 'Телефон', 'device', 60),
        (r_id, 'Сервер', 'device', 70),
        (r_id, 'Роутер/коммутатор', 'device', 80),
        (r_id, 'Память (RAM)', 'component', 110),
        (r_id, 'Накопитель (SSD/HDD)', 'component', 120),
        (r_id, 'Процессор', 'component', 130),
        (r_id, 'Кулер', 'component', 140),
        (r_id, 'Материнская плата', 'component', 150),
        (r_id, 'Видеокарта', 'component', 160),
        (r_id, 'Блок питания', 'component', 170),
        (r_id, 'Корпус', 'component', 180),
        (r_id, 'USB-контроллер', 'component', 190),
        (r_id, 'Клавиатура', 'periphery', 210),
        (r_id, 'Мышь', 'periphery', 220),
        (r_id, 'Наушники', 'periphery', 230),
        (r_id, 'Веб-камера', 'periphery', 240),
        (r_id, 'ИБП', 'periphery', 250),
        (r_id, 'Сумка', 'periphery', 260),
        (r_id, 'Картридж', 'consumable', 310),
        (r_id, 'Патч-корд (кабель)', 'consumable', 320),
        (r_id, 'Батарея/аккумулятор', 'consumable', 330)
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
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location, purchase_date)
        VALUES (r_id, t_desktop, 'Системный блок бухгалтерии', 'Dell', 'OptiPlex 3080', 'DL-3080-0001', 'INV-1001', 'in_use', 'Кабинет 305', '2024-02-01')
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
        INSERT INTO equipment (realm_id, type_id, name, vendor, model, serial_number, inventory_number, status, location, purchase_date, is_kit)
        VALUES (r_id, t_laptop, 'Ноутбук менеджера', 'Lenovo', 'ThinkPad T14', 'LN-T14-0002', 'INV-1002', 'in_use', 'Кабинет 401', '2024-06-15', true)
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

    -- ─── Приведение к текущей модели ───────────────────────────────────────────
    -- Физически внутри может лежать только комплектующее/расходник (EQ014), а
    -- родителем быть только устройство/периферия (EQ015) — восстанавливаем это
    -- для демо-единиц, если прежняя версия сида или ручные правки увели их в сторону.
    UPDATE equipment SET parent_id = e_desktop, kit_id = NULL
    WHERE realm_id = r_id AND inventory_number IN ('INV-1001-01', 'INV-1001-02');

    UPDATE equipment SET parent_id = e_printer, kit_id = NULL
    WHERE realm_id = r_id AND inventory_number = 'INV-1003-01';

    -- Комплект: это признак is_kit, а члены — логические (своих parent_id у них
    -- нет, иначе физсостав в дереве перебивает членство в комплекте).
    UPDATE equipment SET is_kit = true WHERE id = e_laptop;
    UPDATE equipment SET parent_id = NULL WHERE id IN (e_mouse, e_bag);

    -- ─── Состав комплекта (kit_compositions + денормализованный kit_id) ────────
    -- Единица входит только в один комплект, поэтому чужое открытое членство
    -- мыши/сумки закрываем (как CloseOpenByComponent).
    UPDATE kit_compositions SET ended_at = NOW()
    WHERE component_id IN (e_mouse, e_bag) AND ended_at IS NULL AND kit_id <> e_laptop;

    INSERT INTO kit_compositions (kit_id, component_id, started_by, reason)
    SELECT e_laptop, e_mouse, u_id, 'сборка комплекта'
    WHERE NOT EXISTS (
        SELECT 1 FROM kit_compositions WHERE kit_id = e_laptop AND component_id = e_mouse AND ended_at IS NULL
    );

    INSERT INTO kit_compositions (kit_id, component_id, started_by, reason)
    SELECT e_laptop, e_bag, u_id, 'сборка комплекта'
    WHERE NOT EXISTS (
        SELECT 1 FROM kit_compositions WHERE kit_id = e_laptop AND component_id = e_bag AND ended_at IS NULL
    );

    -- kit_id — денормализация текущего комплекта у члена (её ставит OpenComposition);
    -- по ней карточка показывает «В составе комплекта».
    UPDATE equipment SET kit_id = e_laptop WHERE id IN (e_mouse, e_bag);

    -- ─── Пример истории выдачи: системный блок на активного пользователя ───────
    -- Assign пропагирует выдачу на весь состав, поэтому держателя и статус
    -- выставляем на всё поддерево, а не только на сам системный блок.
    IF u_id IS NOT NULL THEN
        FOR v_eq IN
            WITH RECURSIVE subtree AS (
                SELECT id FROM equipment WHERE id = e_desktop
                UNION ALL
                SELECT e.id FROM equipment e JOIN subtree s ON e.parent_id = s.id
            )
            SELECT id FROM subtree
        LOOP
            UPDATE equipment
            SET holder_user_id = u_id, holder_name = NULL, location = 'Кабинет 305', status = 'in_use'
            WHERE id = v_eq;

            INSERT INTO equipment_assignments (equipment_id, user_id, site_id, location, assigned_by, comment)
            SELECT v_eq, u_id, NULL, 'Кабинет 305', u_id, 'Плановое закрепление'
            WHERE NOT EXISTS (
                SELECT 1 FROM equipment_assignments a
                WHERE a.equipment_id = v_eq AND a.unassigned_at IS NULL
            );
        END LOOP;
    END IF;
END $$;
