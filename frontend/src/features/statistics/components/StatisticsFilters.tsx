import { useMemo, useState, type FC } from 'react'
import { Autocomplete, Box, Button, Popover as MuiPopover, TextField, Typography } from '@mui/material'
import { FilterIcon, FilterX } from 'lucide-mui'

import { FilterChip } from '@/components/FilterChip'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import type { IStatisticsRefinements } from '../types/statistics'

interface Option {
	id: string
	label: string
}

interface Props {
	value: IStatisticsRefinements
	onChange: (patch: Partial<IStatisticsRefinements>) => void
	onReset: () => void
	// assignee/group имеют смысл только для управленческих ролей: у рядового
	// пользователя срез и так ограничен его собственными заявками.
	isManager: boolean
}

interface FilterSelectProps {
	options: Option[]
	values: string[]
	onChange: (ids: string[]) => void
}

const FilterSelect: FC<FilterSelectProps> = ({ options, values, onChange }) => {
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

interface PopoverProps {
	open: boolean
	anchorEl: HTMLElement | null
	onClose: () => void
	initial: IStatisticsRefinements
	onApply: (patch: Partial<IStatisticsRefinements>) => void
	isManager: boolean
	assigneeOptions: Option[]
	categoryOptions: Option[]
	groupOptions: Option[]
	siteOptions: Option[]
}

const sectionSx = { fontSize: 12, fontWeight: 600, color: '#57606a', textTransform: 'uppercase' as const, mb: 0.75 }
const sectionBoxSx = { mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }

const FilterPopover: FC<PopoverProps> = ({
	open,
	anchorEl,
	onClose,
	initial,
	onApply,
	isManager,
	assigneeOptions,
	categoryOptions,
	groupOptions,
	siteOptions,
}) => {
	const [local, setLocal] = useState(() => ({
		assigneeId: initial.assigneeId,
		categoryId: initial.categoryId,
		groupId: initial.groupId,
		siteId: initial.siteId,
	}))

	const update = <K extends keyof typeof local>(key: K, value: (typeof local)[K]) => {
		setLocal(prev => ({ ...prev, [key]: value }))
	}

	const handleApply = () => {
		onApply({ ...local })
		onClose()
	}

	const handleReset = () => {
		setLocal({ assigneeId: [], categoryId: [], groupId: [], siteId: [] })
	}

	return (
		<MuiPopover
			open={open}
			anchorEl={anchorEl}
			onClose={onClose}
			anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
			transformOrigin={{ vertical: 'top', horizontal: 'right' }}
			slotProps={{
				paper: {
					sx: { borderRadius: '10px', boxShadow: '0 10px 30px rgba(0,0,0,0.12)', mt: 0.5 },
				},
			}}
		>
			<Box sx={{ width: 380, p: 2 }}>
				{isManager && (
					<Box sx={sectionBoxSx}>
						<Typography sx={sectionSx}>Исполнитель</Typography>
						<FilterSelect
							options={assigneeOptions}
							values={local.assigneeId}
							onChange={ids => update('assigneeId', ids)}
						/>
					</Box>
				)}

				<Box sx={sectionBoxSx}>
					<Typography sx={sectionSx}>Категория</Typography>
					<FilterSelect
						options={categoryOptions}
						values={local.categoryId}
						onChange={ids => update('categoryId', ids)}
					/>
				</Box>

				{isManager && (
					<Box sx={sectionBoxSx}>
						<Typography sx={sectionSx}>Группа</Typography>
						<FilterSelect options={groupOptions} values={local.groupId} onChange={ids => update('groupId', ids)} />
					</Box>
				)}

				<Box sx={{ mb: 2 }}>
					<Typography sx={sectionSx}>Площадка</Typography>
					<FilterSelect options={siteOptions} values={local.siteId} onChange={ids => update('siteId', ids)} />
				</Box>

				<Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1, mt: 2, pt: 2, borderTop: '1px solid #eaeef2' }}>
					<Button
						onClick={handleReset}
						sx={{
							color: '#2f81f7',
							textTransform: 'none',
							p: 0,
							minWidth: 'auto',
							'&:hover': { textDecoration: 'underline', bgcolor: 'transparent' },
						}}
					>
						Сбросить всё
					</Button>
					<Button variant='contained' onClick={handleApply} sx={{ textTransform: 'none', borderRadius: '6px' }}>
						Применить
					</Button>
				</Box>
			</Box>
		</MuiPopover>
	)
}

export const StatisticsFilters: FC<Props> = ({ value, onChange, onReset, isManager }) => {
	const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

	const { data: usersData } = useGetRealmUsersQuery('executors')
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: groupsData } = useGetAllGroupsQuery()
	const { data: sitesData } = useGetAllSitesQuery()

	const assigneeOptions = useMemo<Option[]>(
		() =>
			(usersData?.data ?? [])
				.filter(u => !u.isSystem)
				.map(u => ({ id: u.id, label: `${u.lastName} ${u.firstName} (${u.username})` })),
		[usersData],
	)
	const categoryOptions = useMemo<Option[]>(
		() => (categoriesData?.data ?? []).map(c => ({ id: c.id, label: c.name })),
		[categoriesData],
	)
	const groupOptions = useMemo<Option[]>(
		() => (groupsData?.data ?? []).map(g => ({ id: g.id, label: g.name })),
		[groupsData],
	)
	const siteOptions = useMemo<Option[]>(
		() => (sitesData?.data ?? []).map(s => ({ id: s.id, label: s.name })),
		[sitesData],
	)

	const activeCount = useMemo(
		() =>
			[value.assigneeId, value.categoryId, value.groupId, value.siteId].filter(ids => ids.length > 0).length,
		[value],
	)

	const chips = useMemo(() => {
		const items: { key: string; prefix: string; values: string[]; onClear: () => void }[] = []
		const names = (ids: string[], options: Option[]) => ids.map(id => options.find(o => o.id === id)?.label ?? id)

		if (value.assigneeId.length) {
			items.push({
				key: 'assigneeId',
				prefix: 'Исполнитель',
				values: names(value.assigneeId, assigneeOptions),
				onClear: () => onChange({ assigneeId: [] }),
			})
		}
		if (value.categoryId.length) {
			items.push({
				key: 'categoryId',
				prefix: 'Категория',
				values: names(value.categoryId, categoryOptions),
				onClear: () => onChange({ categoryId: [] }),
			})
		}
		if (value.groupId.length) {
			items.push({
				key: 'groupId',
				prefix: 'Группа',
				values: names(value.groupId, groupOptions),
				onClear: () => onChange({ groupId: [] }),
			})
		}
		if (value.siteId.length) {
			items.push({
				key: 'siteId',
				prefix: 'Площадка',
				values: names(value.siteId, siteOptions),
				onClear: () => onChange({ siteId: [] }),
			})
		}
		return items
	}, [value, assigneeOptions, categoryOptions, groupOptions, siteOptions, onChange])

	return (
		<>
			<Button
				variant='outlined'
				color='inherit'
				onClick={e => setAnchorEl(e.currentTarget)}
				sx={{
					flexShrink: 0,
					height: 40,
					px: 1.5,
					bgcolor: 'background.paper',
					borderColor: 'divider',
					borderRadius: 2,
					textTransform: 'none',
					whiteSpace: 'nowrap',
					color: activeCount > 0 ? 'primary.main' : 'text.primary',
					'&:hover': { bgcolor: 'background.paper', borderColor: 'primary.main' },
				}}
			>
				<FilterIcon sx={{ fontSize: 18, mr: 1 }} />
				Фильтры
				{activeCount > 0 && (
					<Box
						component='span'
						sx={{
							ml: 0.75,
							bgcolor: '#2f81f7',
							color: '#fff',
							borderRadius: '10px',
							px: 0.75,
							py: 0.125,
							fontSize: '0.75rem',
							fontWeight: 600,
							lineHeight: 1.4,
						}}
					>
						{activeCount}
					</Box>
				)}
			</Button>

			<FilterPopover
				key={String(Boolean(anchorEl))}
				open={Boolean(anchorEl)}
				anchorEl={anchorEl}
				onClose={() => setAnchorEl(null)}
				initial={value}
				onApply={onChange}
				isManager={isManager}
				assigneeOptions={assigneeOptions}
				categoryOptions={categoryOptions}
				groupOptions={groupOptions}
				siteOptions={siteOptions}
			/>

			{chips.length > 0 && (
				<Box
					sx={{
						flexBasis: '100%',
						mt: 1.5,
						display: 'flex',
						flexWrap: 'wrap',
						alignItems: 'center',
						gap: 1,
					}}
				>
					{chips.map(chip => (
						<FilterChip key={chip.key} prefix={chip.prefix} values={chip.values} onClear={chip.onClear} />
					))}
					<Button
						size='small'
						color='inherit'
						startIcon={<FilterX sx={{ fontSize: 16 }} />}
						onClick={onReset}
						sx={{ textTransform: 'none', color: '#6b7280' }}
					>
						Сбросить
					</Button>
				</Box>
			)}
		</>
	)
}