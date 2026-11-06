import { useMemo } from 'react'
import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'

import type { ICategory } from '@/features/categories/types/category'
import type { ISite } from '@/features/sites/types/site'
import type { FormValues } from './types'
import type { IEquipment } from '@/features/equipment/types/equipment'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { PermRules } from '@/features/access/constants/permissions'
import { useGetAllEquipmentQuery } from '@/features/equipment/equipmentApiSlice'
import { EquipmentTypeIcon } from '@/features/equipment/components/EquipmentTypeIcon'
import { SectionCard } from './SectionCard'
import { fieldLabelSx } from './styles'

type Props = {
	number?: number
	categories: ICategory[]
	sites: ISite[]
}

// Блок списка: раздел либо одиночная категория без раздела.
type CategoryGroup = {
	key: string
	title: string | null
	items: ICategory[]
}

// Блок списка оборудования: тип (группа) и единицы/комплекты внутри него.
type EquipmentGroup = {
	key: string
	title: string
	items: IEquipment[]
}

const isRootCategory = (category: ICategory) => !category.categoryGroupId

// Ключ группы. У раздела — id раздела (все его категории в одном блоке, чтобы
// заголовок не повторялся), у категории без раздела — её собственный id: таких
// категорий может быть несколько, и каждая идёт отдельным пунктом без заголовка.
const groupKeyOf = (category: ICategory) =>
	category.categoryGroupId ? `section:${category.categoryGroupId}` : `root:${category.id}`

// Стили пунктов списка. `&&&` повышает специфичность: MUI задаёт отступы и
// подсветку правилом `.MuiAutocomplete-listbox .MuiAutocomplete-option` (два
// класса), поэтому обычного sx не хватает — он весит меньше и проигрывает.
//
// Поведение одинаково у всех пунктов: наведение (и подсветка с клавиатуры) —
// лёгкий серый фон, выбранный пункт — синий фон и синяя полоса слева. Границу
// между блоками рисует только разделитель группы, поэтому у пунктов нет своих
// рамок: несколько корневых категорий подряд шли бы с зазорами между линиями.
const optionSx = (isRoot: boolean) => ({
	'&&&': {
		py: isRoot ? '10px' : '8px',
		pr: '16px',
		// 3px занимает полоса выделения, поэтому текст встаёт на 16px (как у
		// заголовков разделов) у корневых пунктов и на 32px у вложенных.
		pl: isRoot ? '13px' : '29px',
		borderLeft: '3px solid transparent',
	},
	'&&&:hover': { bgcolor: '#f3f4f6' },
	'&&&.Mui-focused': { bgcolor: '#f3f4f6' },
	'&&&[aria-selected="true"]': { bgcolor: '#eff6ff', borderLeftColor: '#2563eb' },
	'&&&[aria-selected="true"]:hover': { bgcolor: '#eff6ff', borderLeftColor: '#2563eb' },
	'&&&[aria-selected="true"].Mui-focused': { bgcolor: '#eff6ff', borderLeftColor: '#2563eb' },
})

// Ключ группы оборудования: у единицы — id типа, у комплекта — id комплекта
// (у комплекта, в отличие от единицы, нет типа). Заголовок группы — тип или имя
// комплекта.
const equipmentGroupKey = (equipment: IEquipment) =>
	equipment.typeId ? `type:${equipment.typeId}` : `kit:${equipment.kitId ?? equipment.id}`
const equipmentGroupTitle = (equipment: IEquipment) => equipment.typeName || equipment.name

export const CategoryAndSiteSection = ({ number = 1, categories, sites }: Props) => {
	const { control } = useFormContext<FormValues>()
	const canSeeEquipment = useCheckPermissions({ allOf: [PermRules.Equipment.Read] })

	const { data: equipmentData } = useGetAllEquipmentQuery(
		{ status: 'in_use', limit: 500, excludeKits: true },
		{ skip: !canSeeEquipment },
	)

	const equipment = useMemo(
		() => (!canSeeEquipment ? [] : (equipmentData?.data ?? [])) as IEquipment[],
		[equipmentData, canSeeEquipment],
	)

	// Порядок списка — глобальный порядок категорий, заданный админом
	// (categories.sort_order). Разделы в этом порядке не участвуют: блок
	// раздела стоит там, где стоит его первая категория. Категория без раздела
	// — отдельный блок из одного пункта, поэтому несколько таких пунктов могут
	// стоять в разных местах, в том числе между разделами. Список сплющивается
	// именно так, потому что MUI группирует по непрерывным сериям одинакового
	// `groupBy`: иначе раздел разорвался бы на несколько заголовков.
	const groups = useMemo(() => {
		const byKey = new Map<string, CategoryGroup>()
		const result: CategoryGroup[] = []
		categories.forEach(category => {
			const key = groupKeyOf(category)
			const existing = byKey.get(key)
			if (existing) {
				existing.items.push(category)
				return
			}
			const group: CategoryGroup = {
				key,
				title: isRootCategory(category) ? null : category.categoryGroup?.name || '',
				items: [category],
			}
			byKey.set(key, group)
			result.push(group)
		})
		return result
	}, [categories])

	const orderedCategories = useMemo(() => groups.flatMap(group => group.items), [groups])

	// Индекс нужен только разделителю: у первого блока линии нет.
	const groupIndexes = useMemo(() => new Map(groups.map((group, index) => [group.key, index])), [groups])

	// Группы оборудования — по типу (единица/комплект): блоки с заголовками
	// такими же, как у категорий (раздел → тип), плюс разделители между блоками.
	const equipmentGroups = useMemo(() => {
		const byKey = new Map<string, EquipmentGroup>()
		const order: string[] = []
		equipment.forEach(item => {
			const key = equipmentGroupKey(item)
			let group = byKey.get(key)
			if (!group) {
				group = { key, title: equipmentGroupTitle(item), items: [] }
				byKey.set(key, group)
				order.push(key)
			}
			group.items.push(item)
		})
		return order.map(k => byKey.get(k)!)
	}, [equipment])

	const equipmentGroupIndexes = useMemo(
		() => new Map(equipmentGroups.map((group, index) => [group.key, index])),
		[equipmentGroups],
	)

	return (
		<SectionCard
			number={number}
			title='Что случилось и где'
			subtitle='Выберите категорию и площадку — это поможет быстрее направить заявку нужной группе'
		>
			<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
				<Box>
					<Typography variant='caption' sx={fieldLabelSx}>
						Категория{' '}
						<Typography component='span' color='error'>
							*
						</Typography>
					</Typography>
					<Controller
						control={control}
						name='categoryId'
						rules={{ required: 'Обязательное поле' }}
						render={({ field, fieldState }) => (
							<Autocomplete
								options={orderedCategories}
								getOptionLabel={o => o.name}
								getOptionKey={o => o.id}
								value={categories.find(c => c.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? '')}
								noOptionsText='Нет категорий'
								groupBy={groupKeyOf}
								renderGroup={params => {
									const index = groupIndexes.get(params.group) ?? -1
									const title = index >= 0 ? groups[index].title : null
									const isRoot = title === null
									const prev = index > 0 ? groups[index - 1] : null
									const isPrevRoot = prev?.title === null
									return (
										<Box component='li' key={params.key}>
											{/* Линия одна на стык блоков — и перед разделом, и перед
											    одиночной категорией без раздела. Между несколькими
											    одиночными подряд линию не рисуем, чтобы их не разрывать. */}
											{index > 0 && !(isRoot && isPrevRoot) && (
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
				</Box>

				<Box>
					<Typography variant='caption' sx={fieldLabelSx}>
						Площадка{' '}
						<Typography component='span' color='error'>
							*
						</Typography>
					</Typography>
					<Controller
						control={control}
						name='siteId'
						rules={{ required: 'Обязательное поле' }}
						render={({ field, fieldState }) => (
							<Autocomplete
								options={sites}
								getOptionLabel={o => o.name}
								getOptionKey={o => o.id}
								value={sites.find(s => s.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? '')}
								noOptionsText='Нет площадок'
								renderOption={(props, option) => {
									const { key, ...rest } = props
									return (
										<Box component='li' key={key} {...rest}>
											<Box sx={{ minWidth: 0 }}>
												<Typography sx={{ fontSize: '0.8125rem', fontWeight: 700 }}>
													{option.name}
												</Typography>
												{option.address && (
													<Typography
														sx={{
															fontSize: '0.75rem',
															color: '#4B5563',
															overflow: 'hidden',
															textOverflow: 'ellipsis',
															whiteSpace: 'nowrap',
														}}
													>
														{option.address}
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
										placeholder='Выберите площадку'
									/>
								)}
							/>
						)}
					/>
				</Box>
				{canSeeEquipment && (
					<Box sx={{ gridColumn: { xs: '1 / -1', sm: '1 / -1' } }}>
						<Typography variant='caption' sx={fieldLabelSx}>
							Оборудование
						</Typography>
						<Controller
							control={control}
							name='equipmentId'
							render={({ field }) => (
								<Autocomplete
									options={equipment}
									groupBy={(o: IEquipment) => equipmentGroupKey(o)}
									getOptionLabel={o => o.name}
									getOptionKey={o => o.id}
									value={equipment.find(e => e.id === field.value) ?? null}
									onChange={(_, value) => field.onChange(value?.id ?? null)}
									noOptionsText='Нет оборудования'
									filterOptions={(options, params) => {
										const q = params.inputValue.trim().toLowerCase()
										if (!q) return options
										return options.filter(
											o =>
												o.name.toLowerCase().includes(q) ||
												(o.inventoryNumber ?? '').toLowerCase().includes(q),
										)
									}}
									renderGroup={params => {
										const index = equipmentGroupIndexes.get(params.group) ?? -1
										const group = equipmentGroups[index]
										if (!group) {
											return (
												<Box component='li' key={params.key}>
													{params.children}
												</Box>
											)
										}
										return (
											<Box component='li' key={params.key}>
												{index > 0 && (
													<Box sx={{ height: '1px', bgcolor: 'divider', mx: 2, my: '4px' }} />
												)}
												<Box
													sx={{
														display: 'flex',
														alignItems: 'center',
														gap: 1,
														position: 'sticky',
														top: 0,
														zIndex: 1,
														bgcolor: 'background.paper',
														px: 1,
														py: 0.5,
														mb: 0.5,
													}}
												>
													<EquipmentTypeIcon
														isKit={group.items[0].isKit}
														icon={group.items[0].typeIcon}
														typeName={group.items[0].typeName}
														sx={{ fontSize: 16 }}
													/>
													<Typography
														sx={{
															fontSize: '11px',
															fontWeight: 700,
															textTransform: 'uppercase',
															letterSpacing: '0.05em',
															color: '#6b7280',
														}}
													>
														{group.title}
													</Typography>
												</Box>
												<Box component='ul' sx={{ p: 0 }}>
													{params.children}
												</Box>
											</Box>
										)
									}}
									renderOption={(props, option) => {
										const { key, ...rest } = props
										return (
											<Box component='li' key={key} {...rest}>
												<Typography sx={{ fontSize: '14px', fontWeight: 500, ml: 1 }}>
													{option.name}
													{(option.inventoryNumber || option.serialNumber) && (
														<Typography
															variant='caption'
															sx={{
																ml: 1,
																color: '#9ca3af',
																overflow: 'hidden',
																textOverflow: 'ellipsis',
																whiteSpace: 'nowrap',
															}}
														>
															({option.inventoryNumber || option.serialNumber})
														</Typography>
													)}
												</Typography>
											</Box>
										)
									}}
									renderInput={params => (
										<TextField {...params} size='small' placeholder='Выберите оборудование' />
									)}
								/>
							)}
						/>
					</Box>
				)}
			</Box>
		</SectionCard>
	)
}
