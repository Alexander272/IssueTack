import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'
import { Building2, Layers } from 'lucide-mui'
import type { ICategory } from '@/features/categories/types/category'
import type { ISite } from '@/features/sites/types/site'
import { SectionCard } from './SectionCard'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'

type Props = {
	categories: ICategory[]
	sites: ISite[]
}

export const CategoryAndSiteSection = ({ categories, sites }: Props) => {
	const { control } = useFormContext<FormValues>()

	return (
		<SectionCard
			number={1}
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
								value={categories.find(c => c.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? '')}
								noOptionsText='Нет категорий'
								renderOption={(props, option) => (
									<Box component='li' {...props}>
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
											<Typography sx={{ fontSize: '0.8125rem', fontWeight: 500 }}>{option.name}</Typography>
											{option.description && (
												<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
													{option.description}
												</Typography>
											)}
										</Box>
									</Box>
								)}
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
								value={sites.find(s => s.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? '')}
								noOptionsText='Нет площадок'
								renderOption={(props, option) => (
									<Box component='li' {...props}>
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
											<Typography sx={{ fontSize: '0.8125rem', fontWeight: 500 }}>{option.name}</Typography>
											{option.address && (
												<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
													{option.address}
												</Typography>
											)}
										</Box>
									</Box>
								)}
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
			</Box>
		</SectionCard>
	)
}
