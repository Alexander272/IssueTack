import { API } from '@/app/api'
import { apiSlice, baseQueryWithReAuth } from '@/app/apiSlice'

const contentUrls = new Map<string, string>()

const attachmentsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		uploadAttachment: builder.mutation<{ id: string; message: string }, { entityType: string; entityId: string; file: File }>({
			query: ({ entityType, entityId, file }) => {
				const formData = new FormData()
				formData.append('file', file)
				return {
					url: API.attachments.upload(entityType, entityId),
					method: 'POST',
					body: formData,
				}
			},
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Tasks', id: arg.entityId },
				{ type: 'Tasks', id: 'LIST' },
			],
		}),
		getAttachmentContent: builder.query<{ url: string; size: number }, string>({
			queryFn: async (id, api, extraOptions) => {
				const res = await baseQueryWithReAuth(
					{ url: API.attachments.content(id), responseHandler: (response: Response) => response.blob() },
					api,
					extraOptions,
				)
				if (res.error) return { error: res.error }
				const blob = res.data as Blob
				const url = URL.createObjectURL(blob)
				contentUrls.set(id, url)
				return { data: { url, size: blob.size } }
			},
			onCacheEntryAdded: async (id, { cacheEntryRemoved }) => {
				await cacheEntryRemoved
				const url = contentUrls.get(id)
				if (url) URL.revokeObjectURL(url)
				contentUrls.delete(id)
			},
		}),
	}),
})

export const { useUploadAttachmentMutation, useGetAttachmentContentQuery, useLazyGetAttachmentContentQuery } =
	attachmentsApiSlice
