import type { Priority } from '../../types/task'

export const PRIORITY_DESCRIPTIONS: Record<Priority, string> = {
	low: 'Не мешает работе, можно подождать',
	medium: 'Работа затруднена, есть обходной путь',
	high: 'Работа полностью остановлена',
	urgent: 'Критично, требуется немедленная реакция',
}
