import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ITask, ITaskDTO, ITaskFilter } from './types/task'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

const tasksApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getTasks: builder.query<{ data: ITask[]; total?: number }, ITaskFilter | void>({
			query: filter => ({
				url: API.tickets.base,
				method: 'GET',
				params: filter || undefined,
			}),
			providesTags: [{ type: 'Tasks', id: 'LIST' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		getTaskById: builder.query<{ data: ITask }, string>({
			query: id => ({
				url: API.tickets.byId(id),
				method: 'GET',
			}),
			providesTags: (_result, _error, id) => [{ type: 'Tasks', id }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		createTask: builder.mutation<{ id: string; message: string }, ITaskDTO>({
			query: body => ({
				url: API.tickets.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'Tasks', id: 'LIST' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		updateTask: builder.mutation<{ id: string; message: string }, ITaskDTO>({
			query: body => ({
				url: API.tickets.byId(body.id),
				method: 'PUT',
				body,
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Tasks', id: 'LIST' },
				{ type: 'Tasks', id: arg.id },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		deleteTask: builder.mutation<void, string>({
			query: id => ({
				url: API.tickets.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Tasks', id: 'LIST' }],
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
	useGetTasksQuery,
	useGetTaskByIdQuery,
	useCreateTaskMutation,
	useUpdateTaskMutation,
	useDeleteTaskMutation,
} = tasksApiSlice
