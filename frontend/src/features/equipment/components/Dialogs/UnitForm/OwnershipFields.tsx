import type { FC } from 'react'
import { Autocomplete, Box, MenuItem, TextField } from '@mui/material'
import { Controller, type Control } from 'react-hook-form'

import type { IEquipment, IEquipmentDTO } from '../../../types/equipment'
import type { ISite } from '@/features/sites/types/site'
import type { IUserData } from '@/features/user/types/user'
import { Field } from './Field'
import { inputProps } from './consts'

// Фамилия Имя пользователя для опций выбора (с логином как уточнением).
const userLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full ? `${full} (${u.username})` : u.username
}

type Props = {
	control: Control<IEquipmentDTO>
	sites: ISite[]
	users: IUserData[]
	// Родителей в физическом составе могут быть только устройства/периферия.
	parentCandidates: IEquipment[]
	parentDisabled: boolean
	parentHint?: string
}

// Владение и размещение: площадка, локация, держатель, даты, стоимость и
// физический родитель (только для комплектующего/расходника).
export const OwnershipFields: FC<Props> = ({
	control,
	sites,
	users,
	parentCandidates,
	parentDisabled,
	parentHint,
}) => (
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

		<Field label='Стоимость, ₽'>
			<Controller
				control={control}
				name='cost'
				render={({ field }) => (
					<TextField
						{...field}
						type='number'
						{...inputProps}
						onChange={e => field.onChange(e.target.value === '' ? null : e.target.value)}
					/>
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
						disabled={parentDisabled}
						getOptionLabel={u => u.name}
						value={parentCandidates.find(u => u.id === field.value) ?? null}
						onChange={(_e, value) => field.onChange(value?.id ?? null)}
						renderInput={params => (
							<TextField
								{...params}
								size='small'
								placeholder='Родитель (необязательно)'
								helperText={parentHint}
							/>
						)}
					/>
				)}
			/>
		</Field>
	</Box>
)
