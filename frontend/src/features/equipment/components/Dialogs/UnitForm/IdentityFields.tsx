import type { FC } from 'react'
import { Box, MenuItem, TextField } from '@mui/material'
import { Controller, type Control } from 'react-hook-form'

import type { IEquipmentDTO, IEquipmentType } from '../../../types/equipment'
import { EQUIPMENT_STATUS_OPTIONS } from '../../../types/equipment'
import { Field } from './Field'
import { inputProps } from './consts'

type Props = {
	control: Control<IEquipmentDTO>
	types: IEquipmentType[]
}

// Идентификация единицы: название, тип, статус, производитель, модель и номера.
export const IdentityFields: FC<Props> = ({ control, types }) => (
	<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
		<Field label='Название' required>
			<Controller
				control={control}
				name='name'
				rules={{ required: 'Обязательное поле' }}
				render={({ field, fieldState }) => (
					<TextField
						{...field}
						{...inputProps}
						placeholder='Например: Системный блок'
						error={Boolean(fieldState.error)}
						helperText={fieldState.error?.message}
					/>
				)}
			/>
		</Field>

		<Field label='Тип' required>
			<Controller
				control={control}
				name='typeId'
				rules={{ required: 'Обязательное поле' }}
				render={({ field, fieldState }) => (
					<TextField
						{...field}
						select
						{...inputProps}
						error={Boolean(fieldState.error)}
						helperText={fieldState.error?.message}
					>
						<MenuItem value='' disabled>
							Выберите тип
						</MenuItem>
						{types.map(t => (
							<MenuItem key={t.id} value={t.id}>
								{t.name}
							</MenuItem>
						))}
					</TextField>
				)}
			/>
		</Field>

		<Field label='Статус'>
			<Controller
				control={control}
				name='status'
				render={({ field }) => (
					<TextField {...field} select {...inputProps}>
						{EQUIPMENT_STATUS_OPTIONS.map(s => (
							<MenuItem key={s.value} value={s.value}>
								{s.label}
							</MenuItem>
						))}
					</TextField>
				)}
			/>
		</Field>

		<Field label='Производитель'>
			<Controller
				control={control}
				name='vendor'
				render={({ field }) => <TextField {...field} {...inputProps} placeholder='Например: Lenovo' />}
			/>
		</Field>

		<Field label='Модель'>
			<Controller control={control} name='model' render={({ field }) => <TextField {...field} {...inputProps} />} />
		</Field>

		<Field label='Серийный номер'>
			<Controller
				control={control}
				name='serialNumber'
				render={({ field }) => <TextField {...field} {...inputProps} />}
			/>
		</Field>

		<Field label='Инвентарный номер'>
			<Controller
				control={control}
				name='inventoryNumber'
				render={({ field }) => <TextField {...field} {...inputProps} />}
			/>
		</Field>
	</Box>
)
