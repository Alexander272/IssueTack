import { useMemo } from 'react'
import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
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

type Props = {
	number?: number
	/** Id текущего пользователя для самоназначения исполнителем (только исполнитель). */
	currentUserId?: string | null
}

export const CustomerAndDateSection = ({ number = 4, currentUserId = null }: Props) => {
	const { control } = useFormContext<FormValues>()
	const { data: customersData } = useGetRealmUsersQuery('all')
	const customers = useMemo(() => customersData?.data ?? [], [customersData])
	const ownerId = useWatch({ control, name: 'ownerId' })

	return (
		<SectionCard
			number={number}
			title='Заказчик и сроки'
			subtitle='Укажите заказчика и желаемый срок выполнения. При необходимости назначьте себя исполнителем'
		>
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
							noOptionsText='Нет заказчиков'
						/>
					)}
				/>
			</Box>

			<Box sx={{ mt: 1 }}>
				<Typography variant='caption' sx={fieldLabelSx}>
					Уведомлять
				</Typography>
				<Controller
					control={control}
					name='notifierIds'
					render={({ field }) => (
						<NotifierSelector
							options={customers}
							value={field.value ?? []}
							onChange={field.onChange}
							excludeIds={[currentUserId, ownerId]}
							limitTags={3}
							sx={{
								'& .MuiInputBase-root': {
									height: 40,
									flexWrap: 'nowrap',
									overflow: 'hidden',
									py: '4px !important',
								},
								'& .MuiAutocomplete-tag': {
									height: 24,
									maxWidth: '130px',
									my: 0,
								},
								// Скрываем все чипсы, когда поле в фокусе (идет поиск/ввод текста)
								'& .MuiInputBase-root.Mui-focused .MuiAutocomplete-tag': {
									display: 'none',
								},
								'& .MuiAutocomplete-input': {
									minWidth: '30px !important',
									py: '0 !important',
								},
							}}
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
