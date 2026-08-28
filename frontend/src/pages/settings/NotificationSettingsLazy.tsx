import { lazy } from 'react'

export const NotificationSettings = lazy(() =>
	import('@/features/user/pages/NotificationSettings').then(m => ({ default: m.NotificationSettings })),
)
