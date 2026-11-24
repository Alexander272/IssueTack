-- +goose Up
-- +goose StatementBegin
-- Дата установки связи дубля. StatisticsRepo исключает дубль из отчёта только
-- если связь была установлена до конца периода: заявка, закрытая в прошлом
-- месяце и помеченная дублем сегодня, остаётся в исторических отчётах.
ALTER TABLE IF EXISTS public.tickets
    ADD COLUMN IF NOT EXISTS duplicate_of_set_at TIMESTAMPTZ;

-- Бэкфилл по журналу: dup_linked — момент, когда заявка стала дублем.
-- dup_repointed меняет только основную, его дата браться не должна,
-- а dup_unlinked снимает связь (такие строки здесь не попадают).
UPDATE public.tickets t
SET duplicate_of_set_at = linked.created_at
FROM (
    SELECT DISTINCT ON (al.entity_id) al.entity_id, al.created_at
    FROM public.activity_logs al
    WHERE al.action = 'dup_linked' AND al.entity_type = 'ticket'
    ORDER BY al.entity_id, al.created_at DESC
) linked
WHERE t.id = linked.entity_id
  AND t.duplicate_of IS NOT NULL
  AND t.duplicate_of_set_at IS NULL;

-- Связи без журнала считаем установленными на момент миграции: старые отчёты
-- не переписываются, а с текущего периода дубль уже исключается.
UPDATE public.tickets
SET duplicate_of_set_at = NOW()
WHERE duplicate_of IS NOT NULL AND duplicate_of_set_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tickets_duplicate_of_set_at
    ON public.tickets (duplicate_of_set_at) WHERE duplicate_of IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE IF EXISTS public.tickets
    DROP COLUMN IF EXISTS duplicate_of_set_at;
-- +goose StatementEnd
