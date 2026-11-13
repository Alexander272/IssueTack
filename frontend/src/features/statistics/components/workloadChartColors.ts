import { getAvatarColor } from '@/utils/avatar'
import { getChartColorFromText } from '../utils/color'

export const WORKLOAD_REST_ID = '__rest'
const REST_HEX = '#6b7280'

export interface SeriesColorConfig {
	sat: number
	light: number
	restLight: number
	legendColor: string
}

// Пресеты для 4-серийного графика исполнителей
export const EXECUTOR_SERIES_COLORS = {
	activeTasks: { sat: 75, light: 50, restLight: 65, legendColor: '#475569' },
	activeSubtasks: { sat: 75, light: 64, restLight: 74, legendColor: '#64748b' },
	closedTasks: { sat: 45, light: 76, restLight: 83, legendColor: '#94a3b8' },
	closedSubtasks: { sat: 35, light: 86, restLight: 90, legendColor: '#cbd5e1' },
} as const satisfies Record<string, SeriesColorConfig>

// Пресеты для 2-серийного графика заказчиков
export const OWNER_SERIES_COLORS = {
	active: { sat: 75, light: 50, restLight: 65, legendColor: '#475569' },
	closed: { sat: 45, light: 78, restLight: 85, legendColor: '#94a3b8' },
} as const satisfies Record<string, SeriesColorConfig>

export function getWorkloadCellColor(id: string, config: SeriesColorConfig): string {
	if (id === WORKLOAD_REST_ID) {
		return getChartColorFromText(REST_HEX, 14, config.restLight)
	}
	const avatarHex = getAvatarColor(id)
	return getChartColorFromText(avatarHex, config.sat, config.light)
}
