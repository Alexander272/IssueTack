export interface ICalendarException {
	date: string
	isWorking: boolean
}

export interface IProductionCalendar {
	timezone: string
	years: number[]
	exceptions: ICalendarException[]
}
