import { type FC, useEffect } from 'react'
import { Box } from '@mui/material'
import { FormProvider, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import {
	useGetNotificationSettingsQuery,
	useSaveNotificationSettingsMutation,
	type INotificationSettings,
} from '../../notificationsApiSlice'
import { Fallback } from '@/components/Fallback/Fallback'
import { SettingsActions } from '../SettingsActions'
import { CategoriesCard } from './CategoriesCard'
import { GeneralChannelsCard } from './GeneralChannelsCard'
import { HowItWorksCard } from './HowItWorksCard'

const DEFAULT_SETTINGS: INotificationSettings = { enabled: true, categories: [], groups: [] }

interface SubscriptionsTabProps {
	onDirtyChange?: (dirty: boolean) => void
}

// Вкладка «Подписки»: персональные настройки уведомлений по группам/категориям.
export const SubscriptionsTab: FC<SubscriptionsTabProps> = ({ onDirtyChange }) => {
	const { data, isLoading } = useGetNotificationSettingsQuery()
	const [save, { isLoading: saving }] = useSaveNotificationSettingsMutation()

	const methods = useForm<INotificationSettings>({ defaultValues: DEFAULT_SETTINGS })
	const { reset, handleSubmit, formState } = methods

	useEffect(() => {
		if (data?.data) {
			reset(data.data)
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
					onReset={() => reset(data?.data ?? DEFAULT_SETTINGS)}
				/>

				<GeneralChannelsCard />
				<CategoriesCard />
				<HowItWorksCard />
			</Box>
		</FormProvider>
	)
}