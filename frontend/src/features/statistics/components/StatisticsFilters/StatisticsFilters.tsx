import { useMemo, useState, type FC } from 'react'
import { Box, Button } from '@mui/material'
import { FilterIcon, FilterX } from 'lucide-mui'

import { FilterChip } from '@/components/FilterChip'
import { FilterPopover } from './FilterPopover'
import type { Option, Props } from './types'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'

export const StatisticsFilters: FC<Props> = ({ value, onChange, onReset, isManager }) => {
	const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

	const { data: usersData } = useGetRealmUsersQuery('executors')
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: groupsData } = useGetAllGroupsQuery()
	const { data: sitesData } = useGetAllSitesQuery()

	const assigneeOptions = useMemo<Option[]>(
		() => (usersData?.data ?? []).map(u => ({ id: u.id, label: `${u.lastName} ${u.firstName} (${u.username})` })),
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
		() => [value.assigneeId, value.categoryId, value.groupId, value.siteId].filter(ids => ids.length > 0).length,
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
					'&:hover': { bgcolor: 'background.paper', borderColor: 'divider' },
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
