import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export interface IDeadlineReminders {
	reminders: number[]
}

const deadlineRemindersApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getDeadlineReminders: builder.query<{ data: IDeadlineReminders }, void>({
			query: () => ({
				url: API.notifications.deadlineReminders,
				method: 'GET',
			}),
			providesTags: [{ type: 'DeadlineReminders', id: 'Settings' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),
		saveDeadlineReminders: builder.mutation<{ data: { message: string } }, IDeadlineReminders>({
			query: body => ({
				url: API.notifications.deadlineReminders,
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'DeadlineReminders', id: 'Settings' }],
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

export const { useGetDeadlineRemindersQuery, useSaveDeadlineRemindersMutation } = deadlineRemindersApiSlice