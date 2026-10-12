import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'
import { Building2, Layers, ListTreeIcon, PackageIcon } from 'lucide-mui'
import type { ICategory } from '@/features/categories/types/category'
import type { ISite } from '@/features/sites/types/site'
import type { IEquipment } from '@/features/equipment/types/equipment'
import { useGetAllEquipmentQuery } from '@/features/equipment/equipmentApiSlice'
import { SectionCard } from './SectionCard'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'
import { Stack } from '@mui/system'

type Props = {
	number?: number
	categories: ICategory[]
	sites: ISite[]
}

// Ключ группировки для Autocomplete: раздел категории либо «Без раздела».
const sectionOfCategory = (category: ICategory) => category.categoryGroup?.name || 'Без раздела'

export const CategoryAndSiteSection = ({ number = 1, categories, sites }: Props) => {
	const { control } = useFormContext<FormValues>()
	const { data: equipmentData } = useGetAllEquipmentQuery({ limit: 500 })

	const equipment = (equipmentData?.data ?? []) as IEquipment[]

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
								options={categories}
								getOptionLabel={o => o.name}
								getOptionKey={o => o.id}
								value={categories.find(c => c.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? '')}
								noOptionsText='Нет категорий'
								groupBy={sectionOfCategory}
								renderGroup={params => (
									<Box component='li' key={params.key}>
										<Stack
											direction='row'
											spacing={1}
											sx={{
												alignItems: 'center',
												mb: 0.5,
												position: 'sticky',
												px: 1,
												py: 0.5,
												mt: 0.5,
											}}
										>
											<Box
												sx={{
													width: 30,
													height: 30,
													borderRadius: '8px',
													bgcolor: '#dbeafe',
													color: 'primary.main',
													display: 'flex',
													alignItems: 'center',
													justifyContent: 'center',
													mr: 1.5,
													flexShrink: 0,
												}}
											>
												<ListTreeIcon sx={{ fontSize: 16 }} />
											</Box>
											<Typography
												sx={{ fontSize: '0.75rem', fontWeight: 600, color: 'text.secondary' }}
											>
												{params.group}
											</Typography>
										</Stack>

										<Box component='ul' sx={{ p: 0 }}>
											{params.children}
										</Box>
									</Box>
								)}
								renderOption={(props, option) => {
									const { key, ...rest } = props
									return (
										<Box component='li' key={key} {...rest}>
											<Box
												sx={{
													width: 32,
													height: 32,
													borderRadius: '8px',
													bgcolor: '#dbeafe',
													color: 'primary.main',
													display: 'flex',
													alignItems: 'center',
													justifyContent: 'center',
													mr: 1.5,
													flexShrink: 0,
												}}
											>
												<Layers sx={{ fontSize: 16 }} />
											</Box>
											<Box sx={{ minWidth: 0 }}>
												<Typography sx={{ fontSize: '0.8125rem', fontWeight: 500 }}>
													{option.name}
												</Typography>
												{option.description && (
													<Typography
														sx={{
															fontSize: '0.6875rem',
															color: '#9ca3af',
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
											<Box
												sx={{
													width: 32,
													height: 32,
													borderRadius: '8px',
													bgcolor: '#dbeafe',
													color: 'primary.main',
													display: 'flex',
													alignItems: 'center',
													justifyContent: 'center',
													mr: 1.5,
													flexShrink: 0,
												}}
											>
												<Building2 sx={{ fontSize: 16 }} />
											</Box>
											<Box sx={{ minWidth: 0 }}>
												<Typography sx={{ fontSize: '0.8125rem', fontWeight: 500 }}>
													{option.name}
												</Typography>
												{option.address && (
													<Typography
														sx={{
															fontSize: '0.6875rem',
															color: '#9ca3af',
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
								getOptionLabel={o => o.name}
								getOptionKey={o => o.id}
								value={equipment.find(e => e.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? null)}
								noOptionsText='Нет оборудования'
								renderOption={(props, option) => {
									const { key, ...rest } = props
									return (
										<Box component='li' key={key} {...rest}>
											<Box
												sx={{
													width: 32,
													height: 32,
													borderRadius: '8px',
													bgcolor: '#dbeafe',
													color: 'primary.main',
													display: 'flex',
													alignItems: 'center',
													justifyContent: 'center',
													mr: 1.5,
													flexShrink: 0,
												}}
											>
												<PackageIcon sx={{ fontSize: 16 }} />
											</Box>
											<Box sx={{ minWidth: 0 }}>
												<Typography sx={{ fontSize: '0.8125rem', fontWeight: 500 }}>
													{option.name}
												</Typography>
												{(option.inventoryNumber || option.serialNumber) && (
													<Typography
														sx={{
															fontSize: '0.6875rem',
															color: '#9ca3af',
															overflow: 'hidden',
															textOverflow: 'ellipsis',
															whiteSpace: 'nowrap',
														}}
													>
														{option.inventoryNumber || option.serialNumber}
													</Typography>
												)}
											</Box>
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
			</Box>
		</SectionCard>
	)
}
