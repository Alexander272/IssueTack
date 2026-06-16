import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IAuditLog } from './types/audit'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export const auditApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAuditLogs: builder.query<{ data: IAuditLog[] }, null>({
			query: () => ({
				url: API.audit,
				method: 'GET',
			}),
			providesTags: [{ type: 'AuditLogs', id: 'All' }],
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

export const { useGetAuditLogsQuery } = auditApiSlice
