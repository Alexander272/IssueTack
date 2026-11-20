import { useMemo, type FC } from 'react'
import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'

import type { FormValues } from './types'
import type { IEquipment } from '@/features/equipment/types/equipment'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { PermRules } from '@/features/access/constants/permissions'
import { useGetAllEquipmentQuery } from '@/features/equipment/equipmentApiSlice'
import { EquipmentTypeIcon } from '@/features/equipment/components/EquipmentTypeIcon'
import { SectionCard } from './SectionCard'
import { fieldLabelSx } from './styles'
import { CategoryField, SiteField } from '@/components/FormFields'
import { CategoryModifierField } from './CategoryModifierField'

type Props = {
	number?: number
}

type EquipmentGroup = {
	key: string
	title: string
	items: IEquipment[]
}

const equipmentGroupKey = (equipment: IEquipment) =>
	equipment.typeId ? `type:${equipment.typeId}` : `kit:${equipment.kitId ?? equipment.id}`
const equipmentGroupTitle = (equipment: IEquipment) => equipment.typeName || equipment.name

export const CategoryAndSiteSection: FC<Props> = ({ number = 1 }) => {
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
					<Typography sx={fieldLabelSx}>
						Категория{' '}
						<Typography component='span' color='error'>
							*
						</Typography>
					</Typography>
					<CategoryField name='categoryId' rules={{ required: 'Обязательное поле' }} showDividers />
				</Box>

				<Box>
					<Typography sx={fieldLabelSx}>
						Площадка{' '}
						<Typography component='span' color='error'>
							*
						</Typography>
					</Typography>
					<SiteField name='siteId' rules={{ required: 'Обязательное поле' }} />
				</Box>

				<CategoryModifierField sx={{ gridColumn: { xs: '1 / -1', sm: '1 / -1' } }} />

				{canSeeEquipment && (
					<Box sx={{ gridColumn: { xs: '1 / -1', sm: '1 / -1' } }}>
						<Typography sx={fieldLabelSx}>Оборудование</Typography>
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
