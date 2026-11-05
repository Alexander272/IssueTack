import { useMemo } from 'react'
import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers'
import { Controller, useFormContext } from 'react-hook-form'
import dayjs from 'dayjs'

import type { FormValues } from './types'
import { fieldLabelSx } from './styles'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { DateTextField } from '@/components/DatePicker/DatePicker'
import { CustomerSelector } from './CustomerSelector'
import { SectionCard } from './SectionCard'

type Props = {
	number?: number
	/** Id текущего пользователя для самоназначения исполнителем (только исполнитель). */
	currentUserId?: string | null
}

export const CustomerAndDateSection = ({ number = 4, currentUserId = null }: Props) => {
	const { control } = useFormContext<FormValues>()
	const { data: customersData } = useGetRealmUsersQuery('all')
	const customers = useMemo(() => customersData?.data ?? [], [customersData])

	return (
		<SectionCard
			number={number}
			title='Заказчик и сроки'
			subtitle='Укажите заказчика и желаемый срок выполнения. При необходимости назначьте себя исполнителем'
		>
			<Box>
				<Typography sx={fieldLabelSx}>
					Заказчик{' '}
					<Typography component='span' color='error'>
						*
					</Typography>
				</Typography>
				<Controller
					control={control}
					name='ownerId'
					rules={{ required: 'Обязательное поле' }}
					render={({ field, fieldState }) => (
						<CustomerSelector
							options={customers}
							value={field.value}
							onChange={field.onChange}
							placeholder='Выберите заказчика'
							noOptionsText='Нет заказчиков'
							error={!!fieldState.error}
							helperText={fieldState.error?.message}
						/>
					)}
				/>
			</Box>

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
							slots={{ textField: DateTextField }}
							slotProps={{ textField: { fullWidth: true } }}
						/>
					)}
				/>
			</Box>

			{currentUserId && (
				<Box sx={{ mt: 1 }}>
					<Controller
						control={control}
						name='assigneeId'
						render={({ field }) => (
							<FormControlLabel
								control={
									<Checkbox
										checked={field.value === currentUserId}
										onChange={e => field.onChange(e.target.checked ? currentUserId : null)}
									/>
								}
								label='Назначить меня исполнителем'
							/>
						)}
					/>
				</Box>
			)}
		</SectionCard>
	)
}
