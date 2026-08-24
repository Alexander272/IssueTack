import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'
import type { IActivityLog } from '../../types/activity'

const activityApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getActivityLogs: builder.query<{ data: IActivityLog[] }, string>({
			query: entityId => ({
				url: API.activityLog.byEntity(entityId),
				method: 'GET',
			}),
			providesTags: [{ type: 'ActivityLogs', id: 'LIST' }],
		}),
	}),
})

export const { useGetActivityLogsQuery } = activityApiSlice
