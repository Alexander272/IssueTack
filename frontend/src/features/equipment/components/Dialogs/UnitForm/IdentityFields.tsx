import type { FC } from 'react'
import { Box, MenuItem, TextField } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'

import type { IEquipmentDTO } from '../../../types/equipment'
import { EQUIPMENT_STATUS_OPTIONS, HOLDER_DERIVED_STATUSES } from '../../../types/equipment'
import { useGetAllEquipmentTypesQuery } from '../../../equipmentApiSlice'
import { Field } from './Field'
import { inputProps } from './consts'

// Идентификация единицы: название, тип, статус, производитель, модель и номера.
// Справочник типов секция грузит сама: единственный потребитель — эта форма.
export const IdentityFields: FC = () => {
	const { control } = useFormContext<IEquipmentDTO>()
	const { data } = useGetAllEquipmentTypesQuery()
	const types = data?.data ?? []

	return (
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
						<TextField
							{...field}
							select
							{...inputProps}
							// «На складе»/«В эксплуатации» — следствие выдачи, их выводит
							// бэкенд из держателя. Текущее значение показываем, но выбрать
							// вручную не даём: иначе форма сохранит «В эксплуатации» без
							// держателя — и карточка скажет на неё «На складе».
							helperText='«На складе» и «В эксплуатации» ставит выдача и возврат'
						>
							{EQUIPMENT_STATUS_OPTIONS.map(s => (
								<MenuItem key={s.value} value={s.value} disabled={HOLDER_DERIVED_STATUSES.includes(s.value)}>
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
}
