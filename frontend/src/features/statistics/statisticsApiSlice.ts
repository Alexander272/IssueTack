import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ITask } from '@/features/tasks/types/task'
import type { IStatisticsFilter, IStatisticsTicketsFilter, ITicketStatistics } from './types/statistics'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

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
	}),
})

export const { useGetTicketStatisticsQuery, useGetStatisticsTicketsQuery } = statisticsApiSlice