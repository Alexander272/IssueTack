import { useEffect, useMemo } from 'react'
import { Autocomplete, TextField, type SxProps } from '@mui/material'

import type { IUserData } from '@/features/user/types/user'

type Props = {
	options: IUserData[]
	value: string[]
	onChange: (ids: string[]) => void
	/** Пользователи, которых нельзя выбрать (автор, заказчик и т.п.). */
	excludeIds?: Array<string | null | undefined>
	limitTags?: number
	disabled?: boolean
	sx?: SxProps
}

export const NotifierSelector = ({ options, value, onChange, excludeIds, limitTags, disabled, sx }: Props) => {
	const excluded = useMemo(
		() => new Set((excludeIds ?? []).filter((id): id is string => Boolean(id))),
		[excludeIds],
	)
	const filteredOptions = useMemo(
		() => (excluded.size ? options.filter(u => !excluded.has(u.id)) : options),
		[options, excluded],
	)
	const cleanedValue = useMemo(() => value.filter(id => !excluded.has(id)), [value, excluded])

	useEffect(() => {
		if (cleanedValue.length !== value.length) onChange(cleanedValue)
	}, [cleanedValue, onChange, value.length, value])

	const selected = filteredOptions.filter(u => cleanedValue.includes(u.id))

	return (
		<Autocomplete
			multiple
			options={filteredOptions}
			limitTags={limitTags}
			disableCloseOnSelect
			getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
			value={selected}
			onChange={(_, v) => onChange(v.map(u => u.id))}
			disabled={disabled}
			noOptionsText='Нет пользователей'
			renderInput={params => <TextField {...params} size='small' placeholder='Выберите пользователей' />}
			sx={sx}
		/>
	)
}
