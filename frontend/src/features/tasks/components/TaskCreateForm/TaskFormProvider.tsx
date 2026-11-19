import type { ReactNode } from 'react'

import { TaskFormSettingsContext, type TaskFormSettings } from './taskFormMode'

export const TaskFormProvider = ({
	settings,
	children,
}: {
	settings: TaskFormSettings
	children: ReactNode
}) => <TaskFormSettingsContext.Provider value={settings}>{children}</TaskFormSettingsContext.Provider>
