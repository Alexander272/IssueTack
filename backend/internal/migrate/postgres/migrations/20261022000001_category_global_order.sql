-- +goose Up
-- +goose StatementBegin

-- Глобальный порядок категорий: sort_order — позиция категории в реалме.
-- Раздел больше не задаёт порядок сам: блок раздела стоит там, где стоит его
-- первая категория (см. CategoryRepo.Get), а категории без раздела образуют
-- такой же блок — со своим заголовком в админке, но без заголовка в списках
-- выбора.
ALTER TABLE public.categories ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

-- Бэкфилл: текущий порядок (раздел, затем имя) переносим в sort_order, чтобы
-- списки выбора не «прыгнули» после миграции. Нумерация — в пределах реалма:
-- sort_order — позиция категории внутри своего реалма (1..N).
WITH ordered AS (
    SELECT c.id, ROW_NUMBER() OVER (PARTITION BY c.realm_id ORDER BY COALESCE(cg.sort_order, 2147483647), c.name) AS rn
    FROM public.categories c
    LEFT JOIN public.category_groups cg ON c.category_group_id = cg.id
)
UPDATE public.categories c SET sort_order = ordered.rn FROM ordered WHERE c.id = ordered.id;

-- Порядок разделов выводится из глобального порядка категорий — колонка
-- больше не нужна.
ALTER TABLE public.category_groups DROP COLUMN IF EXISTS sort_order;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE public.category_groups ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

-- Порядок разделов восстанавливаем из минимального sort_order его категорий
-- (раздел стоит там, где стоит его первая категория).
WITH mins AS (
    SELECT cg.id, COALESCE(MIN(c.sort_order), 2147483647) AS mn
    FROM public.category_groups cg
    LEFT JOIN public.categories c ON c.category_group_id = cg.id
    GROUP BY cg.id
)
UPDATE public.category_groups cg SET sort_order = mins.mn FROM mins WHERE cg.id = mins.id;

ALTER TABLE public.categories DROP COLUMN IF EXISTS sort_order;

-- +goose StatementEnd
