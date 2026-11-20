import { useMemo } from 'react'
import { Box, Typography } from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers'
import { Controller, useFormContext, useWatch } from 'react-hook-form'
import dayjs from 'dayjs'

import type { FormValues } from './types'
import { fieldLabelSx } from './styles'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { DateTextField } from '@/components/DatePicker/DatePicker'
import { CustomerSelector } from './CustomerSelector'
import { NotifierSelector } from './NotifierSelector'
import { SectionCard } from './SectionCard'
import { useTaskFormSettings } from './taskFormMode'

type Props = {
	number?: number
	title?: string
	subtitle?: string
}

export const CustomerAndDateSection = ({
	number = 4,
	title = 'Заказчик и сроки',
	subtitle = 'Укажите заказчика и желаемый срок выполнения',
}: Props) => {
	const { creatorId, canEditNotifiers, showOwner, showDueDate, showNotifierFieldLabel, dueDateDisabled } =
		useTaskFormSettings()
	const { control } = useFormContext<FormValues>()
	const { data: customersData } = useGetRealmUsersQuery('all')
	const customers = useMemo(() => customersData?.data ?? [], [customersData])
	const ownerId = useWatch({ control, name: 'ownerId' })
	const excludedNotifierIds = [creatorId, ownerId]

	return (
		<SectionCard number={number} title={title} subtitle={subtitle}>
			{showOwner && (
				<Box>
					<Typography variant='caption' sx={fieldLabelSx}>
						Заказчик
					</Typography>
					<Controller
						control={control}
						name='ownerId'
						render={({ field }) => (
							<CustomerSelector
								options={customers}
								value={field.value}
								onChange={field.onChange}
								placeholder='Не указан'
								noOptionsText='Нет пользователей'
							/>
						)}
					/>
				</Box>
			)}

			{canEditNotifiers && (
				<Box sx={{ mt: 1 }}>
					{showNotifierFieldLabel && (
						<Typography variant='caption' sx={fieldLabelSx}>
							Уведомлять
						</Typography>
					)}

					<Controller
						control={control}
						name='notifierIds'
						render={({ field }) => (
							<NotifierSelector
								options={customers}
								value={field.value ?? []}
								onChange={field.onChange}
								excludeIds={excludedNotifierIds}
							/>
						)}
					/>
				</Box>
			)}

			{showDueDate && (
				<Box sx={{ mt: 1 }}>
					<Typography variant='caption' sx={fieldLabelSx}>
						Срок выполнения
					</Typography>
					<Controller
						control={control}
						name='dueDate'
						render={({ field }) => (
							<DateTimePicker
								value={field.value ? dayjs(field.value) : null}
								onChange={date =>
									field.onChange(date ? (field.value ? date : date.endOf('day')).toISOString() : null)
								}
								disabled={dueDateDisabled}
								slots={{ textField: DateTextField }}
								slotProps={{ textField: { fullWidth: true } }}
							/>
						)}
					/>
				</Box>
			)}
		</SectionCard>
	)
}
