import { type FC } from 'react'
import { Button, IconButton, Stack, TextField } from '@mui/material'
import { PlusIcon, Trash2Icon } from 'lucide-mui'

import type { ISpecPair } from './specs'
import { inputProps } from './consts'

// SpecsEditor — редактор свободных характеристик единицы: плоский список пар
// «название — значение» (specs хранится как Record<string, string>). Список строк
// вместо datasheet-grid выбран ради мобильной формы: две колонки на десктопе,
// на телефоне — значение под названием, без горизонтального скролла.
// Преобразование пар в объект и обратно — в specs.ts (см. specsToPairs).

type Props = {
	pairs: ISpecPair[]
	onChange: (pairs: ISpecPair[]) => void
}

export const SpecsEditor: FC<Props> = ({ pairs, onChange }) => {
	const setPair = (index: number, patch: Partial<ISpecPair>) =>
		onChange(pairs.map((pair, i) => (i === index ? { ...pair, ...patch } : pair)))

	const removePair = (index: number) => onChange(pairs.filter((_, i) => i !== index))

	return (
		<Stack spacing={1}>
			{pairs.map((pair, index) => (
				<Stack key={index} direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ alignItems: { sm: 'center' } }}>
					<TextField
						{...inputProps}
						value={pair.key}
						onChange={e => setPair(index, { key: e.target.value })}
						placeholder='Название (например, Объём памяти)'
						sx={{ flex: 1 }}
					/>
					<TextField
						{...inputProps}
						value={pair.value}
						onChange={e => setPair(index, { value: e.target.value })}
						placeholder='Значение (например, 16 ГБ)'
						sx={{ flex: 1 }}
					/>
					<IconButton
						type='button'
						onClick={() => removePair(index)}
						size='small'
						sx={{ color: 'text.secondary', alignSelf: { xs: 'flex-end', sm: 'center' } }}
					>
						<Trash2Icon sx={{ fontSize: 18 }} />
					</IconButton>
				</Stack>
			))}

			<Button
				type='button'
				onClick={() => onChange([...pairs, { key: '', value: '' }])}
				variant='text'
				size='small'
				startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
				sx={{ textTransform: 'none', alignSelf: 'flex-start' }}
			>
				Добавить характеристику
			</Button>
		</Stack>
	)
}
