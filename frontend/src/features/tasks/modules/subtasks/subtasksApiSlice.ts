import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ISubtask, ISubtaskDTO } from './types/subtask'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

const subtasksApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getSubtasksByTicket: builder.query<{ data: ISubtask[]; total?: number }, string>({
			query: ticketId => ({
				url: API.subtasks.byTicket(ticketId),
				method: 'GET',
			}),
			providesTags: (_result, _error, ticketId) => [{ type: 'Tasks', id: ticketId }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		createSubtask: builder.mutation<{ id: string; message: string }, ISubtaskDTO>({
			query: body => ({
				url: API.subtasks.byTicket(body.ticketId),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _error, arg) => [{ type: 'Tasks', id: arg.ticketId }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		updateSubtask: builder.mutation<{ id: string; message: string }, ISubtaskDTO>({
			query: body => ({
				url: API.subtasks.byId(body.ticketId, body.id),
				method: 'PUT',
				body,
			}),
			invalidatesTags: (_result, _error, arg) => [{ type: 'Tasks', id: arg.ticketId }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		deleteSubtask: builder.mutation<void, { ticketId: string; subtaskId: string }>({
			query: ({ ticketId, subtaskId }) => ({
				url: API.subtasks.byId(ticketId, subtaskId),
				method: 'DELETE',
			}),
			invalidatesTags: (_result, _error, arg) => [{ type: 'Tasks', id: arg.ticketId }],
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

export const {
	useGetSubtasksByTicketQuery,
	useCreateSubtaskMutation,
	useUpdateSubtaskMutation,
	useDeleteSubtaskMutation,
} = subtasksApiSlice
