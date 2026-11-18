import { useEffect, useMemo } from 'react'

import type { IUserData } from '@/features/user/types/user'
import { SelectWithFilter } from '@/components/SelectWithFilter/SelectWithFilter'

type Props = {
	options: IUserData[]
	value: string[]
	onChange: (ids: string[]) => void
	/** Пользователи, которых нельзя выбрать (автор, заказчик и т.п.). */
	excludeIds?: Array<string | null | undefined>
	disabled?: boolean
}

export const NotifierSelector = ({ options, value, onChange, excludeIds, disabled }: Props) => {
	const excluded = useMemo(() => new Set((excludeIds ?? []).filter((id): id is string => Boolean(id))), [excludeIds])
	const userOptions = useMemo(
		() =>
			options.map(u => ({
				id: u.id,
				label: `${u.lastName} ${u.firstName} (${u.username})`,
			})),
		[options],
	)

	const filteredOptions = useMemo(
		() => (excluded.size ? userOptions.filter(u => !excluded.has(u.id)) : userOptions),
		[userOptions, excluded],
	)
	const cleanedValue = useMemo(() => value.filter(id => !excluded.has(id)), [value, excluded])

	useEffect(() => {
		if (cleanedValue.length !== value.length) onChange(cleanedValue)
	}, [cleanedValue, onChange, value.length, value])

	const selected = filteredOptions.filter(u => cleanedValue.includes(u.id))

	return (
		<SelectWithFilter
			placeholder='Выберите пользователей'
			values={selected}
			options={filteredOptions}
			onChange={v => onChange(v.map(u => u.id))}
			disabled={disabled}
		/>
	)
}
