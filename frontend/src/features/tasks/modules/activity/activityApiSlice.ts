import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'
import type { IActivityLog } from '../../types/activity'

const activityApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getActivityLogs: builder.query<{ data: IActivityLog[] }, { entityId?: string; parentId?: string }>({
			query: ({ entityId, parentId }) => {
				const url = parentId
					? API.activityLog.byParent(parentId)
					: entityId
						? API.activityLog.byEntity(entityId)
						: API.activityLog.base
				return {
					url,
					method: 'GET',
				}
			},
			providesTags: [{ type: 'ActivityLogs', id: 'LIST' }],
		}),
	}),
})

export const { useGetActivityLogsQuery } = activityApiSlice
