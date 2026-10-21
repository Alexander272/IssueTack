-- +goose Up
-- +goose StatementBegin

-- Явная связь между строкой выдачи и событием журнала.
--
-- Одна операция (выдача/возврат) записывается дважды: строкой в
-- equipment_assignments (состояние: кто держит единицу, период) и событием в
-- equipment_events (журнал операций). Связь между ними была только временной:
-- клиент сопоставлял их по близости моментов, а моменты — разные факты
-- (assigned_at приходит из формы и может быть задним числом, created_at —
-- момент записи в журнал). Из-за расхождения лента истории показывала одну
-- операцию дважды.
--
-- Теперь связь явная: событие называет строку, которую открыло
-- (assignment_id), и строку, которую закрыло (closed_assignment_id). Ссылок
-- две, потому что одна операция меняет две строки: возврат закрывает выдачу
-- и открывает строку «на складе», переоформление закрывает прежнюю выдачу и
-- открывает новую.

ALTER TABLE public.equipment_events
    ADD COLUMN IF NOT EXISTS assignment_id UUID REFERENCES public.equipment_assignments(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS closed_assignment_id UUID REFERENCES public.equipment_assignments(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_equipment_events_assignment
    ON public.equipment_events(assignment_id) WHERE assignment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_equipment_events_closed_assignment
    ON public.equipment_events(closed_assignment_id) WHERE closed_assignment_id IS NOT NULL;

-- Бэкфилл уже накопленных операций: без него старые выдачи остались бы
-- «несвязанными» и лента показала бы их рядом с событиями — то есть ровно тот
-- дубль, который и чинится.
--
-- Сопоставление — по смыслу операции, а не по близости моментов:
--   * событие «выдано» открывает строку с держателем, «возвращено» — строку без
--     держателя (единица на складе);
--   * строка подходит, если момент события попадает в её период
--     (assigned_at … unassigned_at): событие открыло строку и потому лежит
--     внутри её периода;
--   * из нескольких подходящих берётся самая поздняя по началу (событие
--     открывает строку, а не закрывает её), при равном начале — та, что
--     закрылась раньше (закрытая раньше строка была и открыта раньше).

WITH opened AS (
    SELECT ev.id AS event_id,
           a.id AS row_id,
           ROW_NUMBER() OVER (
               PARTITION BY ev.id
               ORDER BY a.assigned_at DESC, a.unassigned_at ASC NULLS LAST
           ) AS rn
    FROM public.equipment_events ev
    JOIN public.equipment_assignments a
      ON a.equipment_id = ev.equipment_id
     AND ev.created_at >= a.assigned_at - INTERVAL '5 seconds'
     AND ev.created_at <= COALESCE(a.unassigned_at, 'infinity'::timestamptz) + INTERVAL '5 seconds'
    WHERE ev.event_type IN ('assigned', 'unassigned')
      AND (
            (ev.event_type = 'assigned'
             AND (a.user_id IS NOT NULL OR COALESCE(a.holder_name, '') <> ''))
         OR (ev.event_type = 'unassigned'
             AND a.user_id IS NULL
             AND COALESCE(a.holder_name, '') = '')
      )
)
UPDATE public.equipment_events ev
SET assignment_id = opened.row_id
FROM opened
WHERE opened.event_id = ev.id AND opened.rn = 1;

-- Вторая половина связи: строку, которую операция закрыла. Закрытие и запись
-- события идут в одной транзакции, поэтому момент закрытия строки и момент
-- записи события совпадают (с точностью до начала транзакции) — рядом стоящие
-- операции отстоят друг от друга на секунды и ближе не подходят. Строка, уже
-- занятая другим событием как открытая, исключается: закрыть и открыть одну
-- строку одна операция не может. Сопоставление взаимное (ближайшая и для
-- строки, и для события), поэтому одна операция не претендует на две строки.

WITH closed AS (
    SELECT a.id AS row_id,
           ev.id AS event_id,
           ROW_NUMBER() OVER (
               PARTITION BY a.id
               ORDER BY ABS(EXTRACT(EPOCH FROM (ev.created_at - a.unassigned_at)))
           ) AS row_rn,
           ROW_NUMBER() OVER (
               PARTITION BY ev.id
               ORDER BY ABS(EXTRACT(EPOCH FROM (ev.created_at - a.unassigned_at)))
           ) AS ev_rn
    FROM public.equipment_assignments a
    JOIN public.equipment_events ev
      ON ev.equipment_id = a.equipment_id
     AND ev.assignment_id IS DISTINCT FROM a.id
     AND ABS(EXTRACT(EPOCH FROM (ev.created_at - a.unassigned_at))) <= 60
    WHERE a.unassigned_at IS NOT NULL
      AND ev.event_type IN ('assigned', 'unassigned')
)
UPDATE public.equipment_events ev
SET closed_assignment_id = closed.row_id
FROM closed
WHERE closed.event_id = ev.id AND closed.row_rn = 1 AND closed.ev_rn = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS public.idx_equipment_events_closed_assignment;
DROP INDEX IF EXISTS public.idx_equipment_events_assignment;
ALTER TABLE public.equipment_events
    DROP COLUMN IF EXISTS closed_assignment_id,
    DROP COLUMN IF EXISTS assignment_id;

-- +goose StatementEnd
