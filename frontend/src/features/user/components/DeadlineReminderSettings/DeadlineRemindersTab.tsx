import { type FC, useEffect } from 'react'
import { Box } from '@mui/material'
import { FormProvider, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import {
	useGetDeadlineRemindersQuery,
	useSaveDeadlineRemindersMutation,
	type IDeadlineReminders,
} from '../../deadlineRemindersApiSlice'
import { Fallback } from '@/components/Fallback/Fallback'
import { SettingsActions } from '../SettingsActions'
import { HowItWorksCard } from './HowItWorksCard'
import { RemindersCard } from './RemindersCard'
import { sortReminders } from './reminderOptions'

const EMPTY_REMINDERS: IDeadlineReminders = { reminders: [] }

interface DeadlineRemindersTabProps {
	onDirtyChange?: (dirty: boolean) => void
}

// Вкладка «Сроки»: пороги напоминаний исполнителю о приближении дедлайна.
export const DeadlineRemindersTab: FC<DeadlineRemindersTabProps> = ({ onDirtyChange }) => {
	const { data, isLoading } = useGetDeadlineRemindersQuery()
	const [save, { isLoading: saving }] = useSaveDeadlineRemindersMutation()

	const methods = useForm<IDeadlineReminders>({ defaultValues: EMPTY_REMINDERS })
	const { reset, handleSubmit, formState } = methods

	useEffect(() => {
		if (data?.data) {
			reset({ reminders: sortReminders(data.data.reminders ?? []) })
		}
	}, [data, reset])

	useEffect(() => {
		onDirtyChange?.(formState.isDirty)
	}, [formState.isDirty, onDirtyChange])

	const handleSave = handleSubmit(async values => {
		try {
			await save(values)
			reset(values)
			toast.success('Настройки сохранены')
		} catch {
			/* ошибки уже показаны в apiSlice */
		}
	})

	if (isLoading) {
		return <Fallback py={6} />
	}

	return (
		<FormProvider {...methods}>
			<Box component='form' onSubmit={handleSave}>
				<SettingsActions
					dirty={formState.isDirty}
					saving={saving}
					onReset={() => reset({ reminders: sortReminders(data?.data.reminders ?? []) })}
				/>

				<RemindersCard />
				<HowItWorksCard />
			</Box>
		</FormProvider>
	)
}