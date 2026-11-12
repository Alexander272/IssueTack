import { useMemo, type FC } from 'react'
import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'

import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'

type CategoryOption = {
	id: string
	name: string
	description?: string
	categoryGroupId?: string | null
	categoryGroup?: { name: string } | null
}

type Props = {
	name: string
	rules?: object
	showDividers?: boolean
}

const categoryGroupKey = (cat: CategoryOption) =>
	cat.categoryGroupId ? `section:${cat.categoryGroupId}` : `root:${cat.id}`
const isRootCategory = (cat: CategoryOption) => !cat.categoryGroupId

const optionSx = (isRoot: boolean) => ({
	'&&&': {
		py: isRoot ? '10px' : '8px',
		pr: '16px',
		pl: isRoot ? '13px' : '29px',
		borderLeft: '3px solid transparent',
	},
	'&&&:hover': { bgcolor: '#f3f4f6' },
	'&&&.Mui-focused': { bgcolor: '#f3f4f6' },
	'&&&[aria-selected="true"]': { bgcolor: '#eff6ff', borderLeftColor: '#2563eb' },
	'&&&[aria-selected="true"]:hover': { bgcolor: '#eff6ff', borderLeftColor: '#2563eb' },
	'&&&[aria-selected="true"].Mui-focused': { bgcolor: '#eff6ff', borderLeftColor: '#2563eb' },
})

export const CategoryField: FC<Props> = ({ name, rules, showDividers = false }) => {
	const { control } = useFormContext()
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const categories = categoriesData?.data ?? []

	// Порядок групп для разделителей
	const groups = useMemo(() => {
		const byKey = new Map<string, { title: string | null }>()
		const result: { key: string; title: string | null }[] = []
		categoriesData?.data?.forEach(category => {
			const key = categoryGroupKey(category)
			if (byKey.has(key)) return
			const title = !category.categoryGroupId ? null : (category.categoryGroup?.name ?? '')
			byKey.set(key, { title })
			result.push({ key, title })
		})
		return result
	}, [categoriesData])

	return (
		<Controller
			name={name}
			control={control}
			rules={rules}
			render={({ field, fieldState }) => (
				<Autocomplete
					options={categories}
					getOptionLabel={o => o.name}
					getOptionKey={o => o.id}
					value={categories.find(c => c.id === field.value) ?? null}
					onChange={(_e, val) => field.onChange(val?.id ?? '')}
					noOptionsText='Нет категорий'
					groupBy={categoryGroupKey}
					renderGroup={params => {
						const index = groups.findIndex(g => g.key === params.group)
						const title = index >= 0 ? groups[index].title : null
						const isRoot = title === null
						const prev = index > 0 ? groups[index - 1] : null
						const isPrevRoot = prev?.title === null
						return (
							<Box component='li' key={params.key}>
								{showDividers && index > 0 && !(isRoot && isPrevRoot) && (
									<Box sx={{ height: '1px', bgcolor: 'divider', mx: 2, my: '4px' }} />
								)}
								{title && (
									<Typography
										sx={{
											position: 'sticky',
											top: 0,
											zIndex: 1,
											bgcolor: 'background.paper',
											px: '16px',
											pt: '10px',
											pb: '4px',
											fontSize: '11px',
											fontWeight: 700,
											textTransform: 'uppercase',
											letterSpacing: '0.05em',
											color: '#6b7280',
										}}
									>
										{title}
									</Typography>
								)}
								<Box component='ul' sx={{ p: 0 }}>
									{params.children}
								</Box>
							</Box>
						)
					}}
					renderOption={(props, option) => {
						const { key, ...rest } = props
						const isRoot = isRootCategory(option)
						return (
							<Box component='li' key={key} {...rest} sx={optionSx(isRoot)}>
								<Box sx={{ minWidth: 0 }}>
									<Typography
										sx={{
											fontSize: '14px',
											fontWeight: isRoot ? 600 : 500,
											color: 'inherit',
										}}
									>
										{option.name}
									</Typography>
									{option.description && (
										<Typography
											sx={{
												fontSize: '12px',
												color: '#6b7280',
												mt: '2px',
												lineHeight: 1.3,
												overflow: 'hidden',
												textOverflow: 'ellipsis',
												whiteSpace: 'nowrap',
											}}
										>
											{option.description}
										</Typography>
									)}
								</Box>
							</Box>
						)
					}}
					renderInput={params => (
						<TextField
							{...params}
							size='small'
							error={Boolean(fieldState.error)}
							helperText={fieldState.error?.message}
							placeholder='Выберите категорию'
						/>
					)}
				/>
			)}
		/>
	)
}
