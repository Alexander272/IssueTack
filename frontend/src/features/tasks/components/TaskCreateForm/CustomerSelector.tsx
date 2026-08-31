import { Autocomplete, TextField } from '@mui/material'
import type { IUserData } from '@/features/user/types/user'

type Props = {
	options: IUserData[]
	value: string | null
	onChange: (id: string | null) => void
	placeholder?: string
	noOptionsText?: string
	error?: boolean
	helperText?: string
}

export const CustomerSelector = ({
	options,
	value,
	onChange,
	placeholder = 'Не указан',
	noOptionsText = 'Нет пользователей',
	error,
	helperText,
}: Props) => (
	<Autocomplete
		options={options}
		getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
		value={options.find(u => u.id === value) ?? null}
		onChange={(_, v) => onChange(v?.id ?? null)}
		noOptionsText={noOptionsText}
		renderInput={params => (
			<TextField {...params} size='small' placeholder={placeholder} error={error} helperText={helperText} />
		)}
	/>
)
