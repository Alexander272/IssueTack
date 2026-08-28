import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

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
		getAttachmentContent: builder.query<Blob, string>({
			query: id => ({
				url: API.attachments.content(id),
				responseHandler: response => response.blob(),
			}),
		}),
	}),
})

export const { useUploadAttachmentMutation, useGetAttachmentContentQuery, useLazyGetAttachmentContentQuery } =
	attachmentsApiSlice
