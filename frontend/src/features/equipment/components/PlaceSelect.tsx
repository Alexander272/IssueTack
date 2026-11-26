import { useMemo, type FC } from 'react'
import { Autocomplete, Box, TextField, Typography, type FilterOptionsState } from '@mui/material'

import { useGetAllSiteLocationsQuery, useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { CornerDownRightIcon } from 'lucide-mui'

export type PlaceValue = {
	// Выбор площадки целиком — siteId задан, locationId пуст. Выбор локации
	// пишет оба поля; очистка обнуляет оба. Никаких новых справочников из форм
	// equipment создать нельзя — локации заводятся только в карточке площадки.
	siteId: string | null
	locationId: string | null
}

type PlaceOption =
	| { kind: 'site'; key: string; siteId: string; siteName: string }
	| { kind: 'location'; key: string; siteId: string; siteName: string; locationId: string; locationName: string }

type Props = {
	value: PlaceValue
	onChange: (value: PlaceValue) => void
	label?: string
	// Формы оборудования уже рисуют подпись над контролом через Field.
	hideLabel?: boolean
	placeholder?: string
	helperText?: string
	error?: boolean
	disabled?: boolean
}

const byName = (a: string, b: string) => a.localeCompare(b, 'ru')

const optionLabel = (option: PlaceOption) =>
	option.kind === 'site' ? option.siteName : `${option.siteName} · ${option.locationName}`

// Ищем и по площадке, и по расположению: «Склад» должен находить все локации
// склада и саму площадку. Стандартный фильтр MUI смотрит лишь на getOptionLabel,
// которого для дочерних строк было бы недостаточно.
const filterOptions = (opts: PlaceOption[], state: FilterOptionsState<PlaceOption>) => {
	const query = state.inputValue.trim().toLowerCase()
	if (!query) return opts

	// 1. Собираем ID площадок, которые совпали сами ИЛИ у которых совпала хотя бы одна локация
	const matchedSiteIds = new Set<string | number>()
	const directlyMatchedSiteIds = new Set<string | number>()

	opts.forEach(opt => {
		const siteMatch = opt.siteName.toLowerCase().includes(query)
		const locMatch = opt.kind === 'location' && opt.locationName.toLowerCase().includes(query)

		if (siteMatch) {
			matchedSiteIds.add(opt.siteId)
			directlyMatchedSiteIds.add(opt.siteId)
		}
		if (locMatch) {
			matchedSiteIds.add(opt.siteId)
		}
	})

	// 2. Фильтруем исходный массив, сохраняя структуру дерева
	return opts.filter(opt => {
		// Показываем заголовок площадки, если совпала она сама или любая её локация
		if (opt.kind === 'site') {
			return matchedSiteIds.has(opt.siteId)
		}

		// Если пользователь ввёл название самой площадки — показываем все её локации
		if (directlyMatchedSiteIds.has(opt.siteId)) {
			return true
		}

		// Иначе показываем только те локации, которые совпали с поисковым запросом
		return opt.locationName.toLowerCase().includes(query)
	})
}

// Один контрол вместо двух полей «Площадка» и «Расположение». Список заранее
// сплющивается: площадка, сразу её локации, следующая площадка. Отдельного
// пункта «Вся площадка» нет — строка площадки сама выглядит как заголовок
// группы и выбирает площадку целиком обычным кликом по варианту.
export const PlaceSelect: FC<Props> = ({
	value,
	onChange,
	label = 'Место',
	hideLabel = false,
	placeholder = '— Выберите место —',
	helperText,
	error = false,
	disabled = false,
}) => {
	const { data: sitesData, isLoading: sitesLoading } = useGetAllSitesQuery()
	const { data: locationsData, isLoading: locationsLoading } = useGetAllSiteLocationsQuery()
	const options = useMemo<PlaceOption[]>(() => {
		const sites = sitesData?.data ?? []
		const locations = locationsData?.data ?? []
		const bySite = new Map<string, typeof locations>()
		for (const location of locations) {
			const list = bySite.get(location.siteId) ?? []
			list.push(location)
			bySite.set(location.siteId, list)
		}

		const result: PlaceOption[] = []
		for (const site of [...sites].sort((a, b) => byName(a.name, b.name))) {
			result.push({ kind: 'site', key: `site:${site.id}`, siteId: site.id, siteName: site.name })
			for (const location of (bySite.get(site.id) ?? []).sort((a, b) => byName(a.name, b.name))) {
				result.push({
					kind: 'location',
					key: `location:${location.id}`,
					siteId: site.id,
					siteName: site.name,
					locationId: location.id,
					locationName: location.name,
				})
			}
		}
		return result
	}, [sitesData?.data, locationsData?.data])

	const selected = useMemo(() => {
		if (value.locationId) {
			const location = options.find(o => o.kind === 'location' && o.locationId === value.locationId)
			if (location) return location
		}
		if (value.siteId) return options.find(o => o.kind === 'site' && o.siteId === value.siteId) ?? null
		return null
	}, [options, value.locationId, value.siteId])

	return (
		<Autocomplete
			fullWidth
			options={options}
			filterOptions={filterOptions}
			getOptionLabel={optionLabel}
			getOptionKey={option => option.key}
			isOptionEqualToValue={(option, current) => option.key === current.key}
			value={selected}
			disabled={disabled}
			loading={sitesLoading || locationsLoading}
			noOptionsText={options.length ? 'Ничего не найдено' : 'Площадки не добавлены'}
			onChange={(_event, option) => {
				if (!option) onChange({ siteId: null, locationId: null })
				else if (option.kind === 'site') onChange({ siteId: option.siteId, locationId: null })
				else onChange({ siteId: option.siteId, locationId: option.locationId })
			}}
			renderOption={(props, option) => {
				const { key, ...optionProps } = props

				if (option.kind === 'site') {
					return (
						<Box
							component='li'
							key={key}
							{...optionProps}
							sx={{
								'&.MuiAutocomplete-option': {
									px: 2,
									py: 1,
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'space-between',
									gap: 1,
									borderTop: '1px solid',
									borderColor: 'divider',
									'&:first-of-type': { borderTop: 'none' },
								},
							}}
						>
							<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
								<Typography
									sx={{ fontSize: '0.875rem', fontWeight: 600, color: 'text.primary' }}
									noWrap
								>
									{option.siteName}
								</Typography>
							</Box>
						</Box>
					)
				}

				return (
					<Box
						component='li'
						key={key}
						{...optionProps}
						sx={{
							'&.MuiAutocomplete-option': {
								pl: 4.5,
								pr: 2,
								py: 0.75,
								display: 'flex',
								alignItems: 'center',
								gap: 1,
							},
						}}
					>
						<CornerDownRightIcon sx={{ fontSize: 14, color: 'text.disabled', flexShrink: 0 }} />
						<Typography sx={{ fontSize: '0.875rem', color: 'text.secondary' }} noWrap>
							{option.locationName}
						</Typography>
					</Box>
				)
			}}
			renderInput={params => (
				<TextField
					{...params}
					label={hideLabel ? undefined : label}
					placeholder={placeholder}
					error={error}
					helperText={
						helperText ??
						(options.length === 0 && !sitesLoading ? 'Добавьте площадку в справочнике' : undefined)
					}
				/>
			)}
		/>
	)
}
