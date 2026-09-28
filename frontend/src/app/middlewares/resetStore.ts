import { createListenerMiddleware, type TypedStartListening } from '@reduxjs/toolkit'

import { apiSlice } from '@/app/apiSlice'
import type { AppDispatch, RootState } from '@/app/store'
import { STORAGE_KEYS } from '@/constants/storage'
import { resetRealm } from '@/features/realms/realmSlice'
import { resetUser } from '@/features/user/userSlice'

export const resetStoreListener = createListenerMiddleware()

const startResetStoreListener = resetStoreListener.startListening as TypedStartListening<RootState, AppDispatch>

startResetStoreListener({
	actionCreator: resetUser,
	effect: async (_, listenerApi) => {
		await listenerApi.delay(100)
		// Полный сброс состояния на logout/истечении сессии: RTK-кэш, активный
		// реалм в сторе и его копия в localStorage.
		listenerApi.dispatch(resetRealm())
		localStorage.removeItem(STORAGE_KEYS.ActiveRealm)
		listenerApi.dispatch(apiSlice.util.resetApiState())
	},
})
