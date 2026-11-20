import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'
import type { IBaseFetchError } from '@/app/types/error'
import { toast } from 'react-toastify'
import type { IProductionCalendar } from './types'

export const calendarApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getProductionCalendar: builder.query<{ data: IProductionCalendar }, void>({
			query: () => ({
				url: API.calendar.base,
				method: 'GET',
			}),
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

export const { useGetProductionCalendarQuery } = calendarApiSlice
