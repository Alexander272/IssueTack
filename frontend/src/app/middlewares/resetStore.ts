import { createListenerMiddleware, type TypedStartListening } from '@reduxjs/toolkit'

import { apiSlice } from '@/app/apiSlice'
import type { AppDispatch, RootState } from '@/app/store'
import { resetUser } from '@/features/user/userSlice'

export const resetStoreListener = createListenerMiddleware()

const startResetStoreListener = resetStoreListener.startListening as TypedStartListening<RootState, AppDispatch>

startResetStoreListener({
	actionCreator: resetUser,
	effect: async (_, listenerApi) => {
		await listenerApi.delay(100)
		// listenerApi.dispatch(resetDataTableState())
		//TODO reset states
		listenerApi.dispatch(apiSlice.util.resetApiState())
	},
})
