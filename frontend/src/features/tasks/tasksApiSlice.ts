import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IDuplicateLinkResult, IDuplicateRef, IDueDateRequest, ITask, ITaskDTO, ITaskFilter } from './types/task'
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
					toast.error(fetchError.data?.message, { autoClose: false })
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
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getTicketDuplicates: builder.query<{ data: IDuplicateRef[] }, string>({
			query: id => ({
				url: API.tickets.duplicates(id),
				method: 'GET',
			}),
			providesTags: (_result, _error, id) => [
				{ type: 'Tasks', id: 'duplicates:LIST' },
				{ type: 'Tasks', id: `duplicates:${id}` },
			],
		}),

		createTask: builder.mutation<{ id: string; message: string }, ITaskDTO>({
			query: body => ({
				url: API.tickets.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [
				{ type: 'Tasks', id: 'LIST' },
				{ type: 'Statistics', id: 'TICKETS' },
			],
		}),

		updateTask: builder.mutation<{ id: string; message: string }, Partial<ITaskDTO> & { id: string }>({
			query: body => ({
				url: API.tickets.byId(body.id),
				method: 'PUT',
				body,
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Tasks', id: 'LIST' },
				{ type: 'Tasks', id: arg.id },
				{ type: 'Tasks', id: `duplicates:${arg.id}` },
				{ type: 'Tasks', id: 'duplicates:LIST' },
				{ type: 'Statistics', id: 'TICKETS' },
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

		takeTask: builder.mutation<{ id: string; message: string }, string>({
			query: id => ({
				url: API.tickets.take(id),
				method: 'POST',
			}),
			invalidatesTags: (_result, _error, id) => [
				{ type: 'Tasks', id: 'LIST' },
				{ type: 'Tasks', id },
				{ type: 'Statistics', id: 'TICKETS' },
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

		transferTask: builder.mutation<{ id: string; message: string }, { id: string; assigneeId: string }>({
			query: ({ id, assigneeId }) => ({
				url: API.tickets.transfer(id),
				method: 'POST',
				body: { assigneeId },
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Tasks', id: 'LIST' },
				{ type: 'Tasks', id: arg.id },
				{ type: 'Statistics', id: 'TICKETS' },
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

		linkDuplicate: builder.mutation<{ data: IDuplicateLinkResult }, { id: string; primaryId: string }>({
			query: ({ id, primaryId }) => ({
				url: API.tickets.duplicate(id),
				method: 'POST',
				body: { primaryId },
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Tasks', id: 'LIST' },
				{ type: 'Tasks', id: arg.id },
				{ type: 'Tasks', id: arg.primaryId },
				{ type: 'Tasks', id: `duplicates:${arg.id}` },
				{ type: 'Tasks', id: `duplicates:${arg.primaryId}` },
				{ type: 'Tasks', id: 'duplicates:LIST' },
				{ type: 'Statistics', id: 'TICKETS' },
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

		unlinkDuplicate: builder.mutation<{ data: IDuplicateLinkResult }, string>({
			query: id => ({
				url: API.tickets.unlinkDuplicate(id),
				method: 'POST',
			}),
			invalidatesTags: (result, _error, id) => {
				const rootTags: { type: 'Tasks'; id: string }[] = result?.data?.rootId
					? [{ type: 'Tasks', id: `duplicates:${result.data.rootId}` }]
					: []
				return [
					{ type: 'Tasks', id: 'LIST' },
					{ type: 'Tasks', id },
					{ type: 'Tasks', id: `duplicates:${id}` },
					{ type: 'Tasks', id: 'duplicates:LIST' },
					...rootTags,
					{ type: 'Statistics', id: 'TICKETS' },
				]
			},
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		deleteTask: builder.mutation<void, string>({
			query: id => ({
				url: API.tickets.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: (_result, _error, id) => [
				{ type: 'Tasks', id: 'LIST' },
				{ type: 'Tasks', id },
				{ type: 'Tasks', id: `duplicates:${id}` },
				{ type: 'Tasks', id: 'duplicates:LIST' },
				{ type: 'Statistics', id: 'TICKETS' },
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

		getIsSubscribed: builder.query<{ data: { subscribed: boolean } }, string>({
			query: id => ({
				url: API.tickets.subscription(id),
				method: 'GET',
			}),
			providesTags: (_result, _error, id) => [{ type: 'Tasks', id: `sub:${id}` }],
		}),

		getDueDateRequests: builder.query<{ data: IDueDateRequest | null }, string>({
			query: ticketId => ({
				url: API.tickets.dueDateRequests(ticketId),
				method: 'GET',
			}),
			providesTags: (_result, _error, ticketId) => [{ type: 'DueDateRequests', id: ticketId }],
		}),

		createDueDateRequest: builder.mutation<{ data: IDueDateRequest }, { ticketId: string; proposedDueDate: string; reason: string }>({
			query: ({ ticketId, proposedDueDate, reason }) => ({
				url: API.tickets.dueDateRequests(ticketId),
				method: 'POST',
				body: { proposedDueDate, reason },
			}),
			invalidatesTags: (_result, _error, arg) => [{ type: 'DueDateRequests', id: arg.ticketId }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		decideDueDateRequest: builder.mutation<{ data: IDueDateRequest }, { ticketId: string; requestId: string; approved: boolean; decisionComment?: string }>({
			query: ({ ticketId, requestId, ...body }) => ({
				url: API.tickets.dueDateRequestDecide(ticketId, requestId),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'DueDateRequests', id: arg.ticketId },
				{ type: 'Tasks', id: arg.ticketId },
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

		subscribe: builder.mutation<{ data: { message: string } }, string>({
			query: id => ({
				url: API.tickets.subscription(id),
				method: 'POST',
			}),
			invalidatesTags: (_result, _error, id) => [{ type: 'Tasks', id: `sub:${id}` }],
		}),

		unsubscribe: builder.mutation<{ data: { message: string } }, string>({
			query: id => ({
				url: API.tickets.subscription(id),
				method: 'DELETE',
			}),
			invalidatesTags: (_result, _error, id) => [{ type: 'Tasks', id: `sub:${id}` }],
		}),
	}),
})

export const {
	useGetTasksQuery,
	useGetTaskByIdQuery,
	useGetTicketDuplicatesQuery,
	useCreateTaskMutation,
	useUpdateTaskMutation,
	useTakeTaskMutation,
	useTransferTaskMutation,
	useLinkDuplicateMutation,
	useUnlinkDuplicateMutation,
	useDeleteTaskMutation,
	useGetIsSubscribedQuery,
	useSubscribeMutation,
	useUnsubscribeMutation,
	useGetDueDateRequestsQuery,
	useCreateDueDateRequestMutation,
	useDecideDueDateRequestMutation,
} = tasksApiSlice
