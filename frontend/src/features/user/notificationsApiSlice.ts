import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export interface ICategoryNotificationSetting {
	id: string
	newTask: boolean
	status: boolean
	comment: boolean
	overdue: boolean
}

export interface IGroupNotificationSetting {
	id: string
	newTask: boolean
	overdue: boolean
}

export interface INotificationSettings {
	enabled: boolean
	categories: ICategoryNotificationSetting[]
	groups: IGroupNotificationSetting[]
}

export const emptyNotificationSettings = (): INotificationSettings => ({
	enabled: true,
	categories: [],
	groups: [],
})

const notificationsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getNotificationSettings: builder.query<{ data: INotificationSettings }, void>({
			query: () => ({
				url: API.notifications.getSettings,
				method: 'GET',
			}),
			providesTags: [{ type: 'Notifications', id: 'Settings' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),
		saveNotificationSettings: builder.mutation<{ data: { message: string } }, INotificationSettings>({
			query: body => ({
				url: API.notifications.settings,
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'Notifications', id: 'Settings' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),
	}),
})

export const { useGetNotificationSettingsQuery, useSaveNotificationSettingsMutation } = notificationsApiSlice
