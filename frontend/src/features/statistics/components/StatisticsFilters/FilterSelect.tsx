import { useMemo, type FC } from 'react'
import { Autocomplete, TextField } from '@mui/material'

import type { FilterSelectProps } from './types'

export const FilterSelect: FC<FilterSelectProps> = ({ options, values, onChange }) => {
	const selected = useMemo(
		() => values.map(id => options.find(o => o.id === id) ?? { id, label: id }),
		[values, options],
	)

	return (
		<Autocomplete
			multiple
			disableCloseOnSelect
			options={options}
			value={selected}
			onChange={(_, next) => onChange(next.map(o => o.id))}
			getOptionLabel={o => o.label}
			isOptionEqualToValue={(o, v) => o.id === v.id}
			limitTags={2}
			renderInput={params => <TextField {...params} placeholder='Выберите...' />}
			noOptionsText='Нет вариантов'
		/>
	)
}
