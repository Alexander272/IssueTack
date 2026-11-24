import type { FilterValues } from '../components/filters'
import { groupByFieldHidden } from '../constants/taskMaps'

/**
 * clampHiddenFields — очистка сохранённого в localStorage фильтра от полей,
 * которые «обычному пользователю» без групп не показываются.
 *
 * Раньше это правило было продублировано в TaskList и FavoritesPage, причём в
 * TaskList проверка «есть ли что скрывать» считала активным пустой поиск и любой
 * непустой объект, поэтому фильтр сбрасывался почти всегда. Здесь правило одно:
 *
 * - mode='all' (список заявок): безгрупповому скрыты и группировка, и фильтры
 *   целиком, поэтому набор сбрасывается к defaults с выключенной группировкой;
 *   equipmentId сохраняется — он приходит deep-link'ом и не является «фильтром
 *   из UI», который пользователь не видит.
 * - mode='filters' (избранное): скрываются только срок, приоритет и группировка
 *   по ним; поиск/заказчик/площадка/статус остаются рабочими.
 */
export const clampHiddenFields = (
	f: FilterValues,
	{ hasNoGroups, defaults, mode }: { hasNoGroups: boolean; defaults: FilterValues; mode: 'all' | 'filters' },
): FilterValues => {
	if (!hasNoGroups) return f

	if (mode === 'all') {
		const hasVisible =
			f.groupEnabled ||
			f.search !== '' ||
			f.ticketNumber !== undefined ||
			f.ownerId !== undefined ||
			(f.siteIds?.length ?? 0) > 0 ||
			f.dueDateFrom !== undefined ||
			f.dueDateTo !== undefined ||
			(f.priorities?.length ?? 0) > 0 ||
			f.assigneeId !== undefined ||
			(f.statuses?.length ?? 0) > 0 ||
			f.duplicates !== undefined ||
			f.sort !== defaults.sort
		if (!hasVisible) return f
		return { ...defaults, groupEnabled: false, equipmentId: f.equipmentId }
	}

	const hiddenGroup = groupByFieldHidden(f.groupBy, hasNoGroups)
	const hasHidden = f.dueDateFrom !== undefined || f.dueDateTo !== undefined || f.priorities !== undefined || hiddenGroup
	if (!hasHidden) return f
	return {
		...f,
		dueDateFrom: undefined,
		dueDateTo: undefined,
		priorities: undefined,
		groupBy: hiddenGroup ? defaults.groupBy : f.groupBy,
	}
}
