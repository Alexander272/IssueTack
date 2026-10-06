import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IStatisticsFilter, ITicketStatistics } from './types/statistics'
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
	}),
})

export const { useGetTicketStatisticsQuery } = statisticsApiSlice