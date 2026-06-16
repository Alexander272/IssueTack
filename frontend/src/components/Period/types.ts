export type Period = 'today' | 'week' | 'month' | 'quarter' | 'year' | 'custom'
export type Preset = {
	label: string
	value: Period
	type?: 'day' | 'month' | 'year'
	count?: number
	// days?: number
}

export interface DateRange {
	startDate: string
	endDate: string
}
