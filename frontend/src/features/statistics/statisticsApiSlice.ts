import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ITask } from '@/features/tasks/types/task'
import type { IStatisticsFilter, IStatisticsTicketsFilter, ITicketStatistics, KpiCardKey } from './types/statistics'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

interface IKpiTicketsFilter extends IStatisticsFilter {
	cardKey: KpiCardKey
	limit?: number
	offset?: number
}

const statisticsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getTicketStatistics: builder.query<{ data: ITicketStatistics }, IStatisticsFilter>({
			query: filter => ({
				url: API.statistics.tickets,
				method: 'GET',
				params: filter,
			}),
			providesTags: [{ type: 'Statistics', id: 'TICKETS' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		// Список заявок сектора диаграммы (drill-down). Срез считается на бэкенде по
		// тем же период/уточнениям, что и агрегаты, поэтому число в списке совпадает
		// с числом в секции.
		getStatisticsTickets: builder.query<{ data: ITask[]; total?: number }, IStatisticsTicketsFilter>({
			query: params => ({
				url: API.statistics.ticketsList,
				method: 'GET',
				params,
			}),
			providesTags: (_result, _error, arg) => [
				{ type: 'Statistics', id: `TICKETS-${arg.dim}-${arg.id}-${arg.ring ?? 'all'}` },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		// Список заявок для KPI-карточек
		getKpiTickets: builder.query<{ data: ITask[]; total?: number }, IKpiTicketsFilter>({
			query: params => ({
				url: API.statistics.kpiTickets,
				method: 'GET',
				params,
			}),
			providesTags: (_result, _error, arg) => [{ type: 'Statistics', id: `KPI-${arg.cardKey}` }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),
	}),
})

export const { useGetTicketStatisticsQuery, useGetStatisticsTicketsQuery, useGetKpiTicketsQuery } = statisticsApiSlice