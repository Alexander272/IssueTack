import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

import type { IComment } from '../../types/comment'

const commentsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getComments: builder.query<{ data: IComment[]; count: number }, string>({
			query: ticketId => API.comments.byTicket(ticketId),
			providesTags: (_result, _error, ticketId) => [{ type: 'Comments' as const, id: ticketId }],
		}),
		createComment: builder.mutation<{ id: string; message: string }, { ticketId: string; text: string; isInternal: boolean; type?: string }>({
			query: ({ ticketId, ...body }) => ({
				url: API.comments.byTicket(ticketId),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Comments' as const, id: arg.ticketId },
				{ type: 'ActivityLogs' as const },
			],
		}),
		deleteComment: builder.mutation<void, { ticketId: string; commentId: string }>({
			query: ({ ticketId, commentId }) => ({
				url: API.comments.delete(ticketId, commentId),
				method: 'DELETE',
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Comments' as const, id: arg.ticketId },
				{ type: 'ActivityLogs' as const },
			],
		}),
	}),
})

export const { useGetCommentsQuery, useCreateCommentMutation, useDeleteCommentMutation } = commentsApiSlice
