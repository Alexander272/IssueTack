import { useMemo } from 'react'
import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { CustomerSelector } from './CustomerSelector'
import { SectionCard } from './SectionCard'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'

type Props = {
	number?: number
	/** Id текущего пользователя для самоназначения исполнителем (только исполнитель). */
	currentUserId?: string | null
}

export const CustomerSelectionSection = ({ number = 4, currentUserId = null }: Props) => {
	const { control } = useFormContext<FormValues>()
	const { data: customersData } = useGetRealmUsersQuery('customers')
	const customers = useMemo(() => customersData?.data ?? [], [customersData])

	return (
		<SectionCard number={number} title='Заказчик' subtitle='Выберите заказчика, для которого создаётся заявка'>
			<Box>
				<Typography component='div' sx={fieldLabelSx}>
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

			{currentUserId && (
				<Box sx={{ mt: 1.5 }}>
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
