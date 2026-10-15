import { useEffect, type FC } from 'react'
import { Autocomplete, Box, MenuItem, TextField } from '@mui/material'
import { Controller, useFormContext, useWatch } from 'react-hook-form'

import type { IEquipmentDTO } from '../../../types/equipment'
import { canBePart } from '../../../types/equipment'
import { useGetAllEquipmentQuery, useGetAllEquipmentTypesQuery } from '../../../equipmentApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { Field } from './Field'
import { inputProps } from './consts'

// Фамилия Имя пользователя для опций выбора (с логином как уточнением).
const userLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full ? `${full} (${u.username})` : u.username
}

// Владение и размещение: площадка, локация, держатель, даты и физический
// родитель (только для комплектующего/расходника). Справочники и вычисления
// секция держит сама — из формы ей нужен только контекст.
export const OwnershipFields: FC = () => {
	const { control, setValue } = useFormContext<IEquipmentDTO>()
	const selfId = useWatch({ control, name: 'id' })
	const typeId = useWatch({ control, name: 'typeId' })

	const { data: sitesData } = useGetAllSitesQuery()
	const { data: usersData } = useGetRealmUsersQuery('executors')
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 })
	const { data: typesData } = useGetAllEquipmentTypesQuery()

	const sites = sitesData?.data ?? []
	const users = usersData?.data ?? []
	const types = typesData?.data ?? []
	// Родителем в физическом составе может быть только устройство или периферия:
	// комплектующее/расходник не содержит состав, а комплект ведёт состав логически
	// (kit_compositions), не через parent_id.
	const parentCandidates = (unitsData?.data ?? []).filter(
		u => u.id !== selfId && !u.isKit && !canBePart(u.typeKind),
	)
	const selectedType = types.find(t => t.id === typeId)
	const canBePartType = canBePart(selectedType?.kind)

	// В физический состав (parent_id) можно положить только комплектующее или
	// расходник: у остальных родитель недопустим — сбрасываем и блокируем поле.
	useEffect(() => {
		if (selectedType && !canBePart(selectedType.kind)) setValue('parentId', null)
	}, [selectedType, setValue])

	return (
		<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
			<Field label='Площадка'>
				<Controller
					control={control}
					name='siteId'
					render={({ field }) => (
						<TextField {...field} select {...inputProps} value={field.value ?? ''}>
							<MenuItem value=''>Не указана</MenuItem>
							{sites.map(s => (
								<MenuItem key={s.id} value={s.id}>
									{s.name}
								</MenuItem>
							))}
						</TextField>
					)}
				/>
			</Field>

			<Field label='Локация на площадке'>
				<Controller
					control={control}
					name='location'
					render={({ field }) => <TextField {...field} {...inputProps} placeholder='Склад, полка A' />}
				/>
			</Field>

			<Field label='Держатель (сотрудник)'>
				<Controller
					control={control}
					name='holderUserId'
					render={({ field }) => (
						<Autocomplete
							options={users}
							getOptionLabel={userLabel}
							value={users.find(u => u.id === field.value) ?? null}
							onChange={(_e, value) => field.onChange(value?.id ?? null)}
							renderInput={params => <TextField {...params} size='small' placeholder='Выберите сотрудника' />}
						/>
					)}
				/>
			</Field>

			<Field label='Держатель (название)'>
				<Controller
					control={control}
					name='holderName'
					render={({ field }) => (
						<TextField {...field} {...inputProps} placeholder='Или свободный текст' />
					)}
				/>
			</Field>

			<Field label='Дата покупки'>
				<Controller
					control={control}
					name='purchaseDate'
					render={({ field }) => (
						<TextField {...field} type='date' {...inputProps} slotProps={{ inputLabel: { shrink: true } }} />
					)}
				/>
			</Field>

			<Field label='Гарантия до'>
				<Controller
					control={control}
					name='warrantyUntil'
					render={({ field }) => (
						<TextField {...field} type='date' {...inputProps} slotProps={{ inputLabel: { shrink: true } }} />
					)}
				/>
			</Field>

			<Field label='Часть комплекта'>
				<Controller
					control={control}
					name='parentId'
					render={({ field }) => (
						<Autocomplete
							options={parentCandidates}
							disabled={!canBePartType}
							getOptionLabel={u => u.name}
							value={parentCandidates.find(u => u.id === field.value) ?? null}
							onChange={(_e, value) => field.onChange(value?.id ?? null)}
							renderInput={params => (
								<TextField
									{...params}
									size='small'
									placeholder='Родитель (необязательно)'
									helperText={
										selectedType && !canBePartType
											? 'В физический состав можно включить только комплектующее или расходник'
											: undefined
									}
								/>
							)}
						/>
					)}
				/>
			</Field>
		</Box>
	)
}
