import type { IEquipmentTypePeriodicTask } from '../types/equipment'
import { pluralize } from '@/utils/plural'

// periodLabel собирает человекочитаемую подпись периода из годов/месяцев/дней.
// Один вид на все места показа (список шаблонов в настройках и вкладка единицы),
// чтобы «Период» не читался по-разному на соседних экранах.
export const periodLabel = (t: IEquipmentTypePeriodicTask): string => {
	const parts: string[] = []
	if (t.periodYears > 0) parts.push(`${t.periodYears} ${pluralize(t.periodYears, ['год', 'года', 'лет'])}`)
	if (t.periodMonths > 0) parts.push(`${t.periodMonths} ${pluralize(t.periodMonths, ['месяц', 'месяца', 'месяцев'])}`)
	if (t.periodDays > 0) parts.push(`${t.periodDays} ${pluralize(t.periodDays, ['день', 'дня', 'дней'])}`)
	return parts.length > 0 ? parts.join(' ') : 'Без периода'
}
