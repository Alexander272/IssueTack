import { type FC, useEffect } from 'react'
import { Box } from '@mui/material'
import { FormProvider, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import { Fallback } from '@/components/Fallback/Fallback'
import {
	useGetNotificationSettingsQuery,
	useSaveNotificationSettingsMutation,
	type INotificationSettings,
} from '../notificationsApiSlice'

import {
	CategoriesCard,
	GeneralChannelsCard,
	GroupsCard,
	HowItWorksCard,
	SettingsHeader,
} from '../components/NotificationSettings'

const DEFAULT_SETTINGS: INotificationSettings = { enabled: true, categories: [], groups: [] }

export const NotificationSettings: FC = () => {
	const { data, isLoading } = useGetNotificationSettingsQuery()
	const [save, { isLoading: saving }] = useSaveNotificationSettingsMutation()

	const methods = useForm<INotificationSettings>({ defaultValues: DEFAULT_SETTINGS })
	const { reset, handleSubmit, formState } = methods

	useEffect(() => {
		if (data?.data) {
			reset(data.data)
		}
	}, [data, reset])

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
			<Box component='form' onSubmit={handleSave} sx={{ flexGrow: 1, overflow: 'auto', p: 3 }}>
				<SettingsHeader
					dirty={formState.isDirty}
					saving={saving}
					onReset={() => reset(data?.data ?? DEFAULT_SETTINGS)}
				/>

				<GeneralChannelsCard />
				<CategoriesCard />
				<GroupsCard />
				<HowItWorksCard />
			</Box>
		</FormProvider>
	)
}
