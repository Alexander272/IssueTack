import { useState, useMemo, type FC } from 'react'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetAllUsersQuery } from '@/features/user/usersApiSlice'

import { Toolbar } from './Toolbar'
import { Popover } from './Popover'
import { Chips } from './Chips'
import type { FilterValues, TaskFiltersProps } from './types'

export type { FilterValues }
export type { TaskFiltersProps }

export const TaskFilters: FC<TaskFiltersProps> = ({ filters, onChange, onReset }) => {
	const [filterAnchorEl, setFilterAnchorEl] = useState<HTMLElement | null>(null)

	const { data: sitesData } = useGetAllSitesQuery()
	const { data: usersData } = useGetAllUsersQuery()

	const siteOptions = useMemo(() => (sitesData?.data ?? []).map(s => ({ id: s.id, label: s.name })), [sitesData])
	const userOptions = useMemo(
		() => (usersData?.data ?? []).map(u => ({ id: u.id, label: `${u.lastName} ${u.firstName} (${u.username})` })),
		[usersData],
	)

	const activeCount = useMemo(() => {
		let count = 0
		if (filters.ticketNumber) count++
		if (filters.ownerId) count++
		if (filters.siteIds?.length) count++
		if (filters.dueDateFrom || filters.dueDateTo) count++
		if (filters.priorities?.length) count++
		if (filters.assigneeId) count++
		if (filters.statuses?.length) count++
		return count
	}, [filters])

	return (
		<>
			<Toolbar
				search={filters.search}
				groupEnabled={filters.groupEnabled}
				groupBy={filters.groupBy}
				activeCount={activeCount}
				onSearchChange={search => onChange({ search })}
				onGroupChange={option =>
					onChange({ groupBy: option ?? 'category', groupEnabled: option ? true : false })
				}
				onOpenFilter={e => setFilterAnchorEl(e.currentTarget)}
				onReset={onReset}
			/>

			<Popover
				key={String(Boolean(filterAnchorEl))}
				open={Boolean(filterAnchorEl)}
				anchorEl={filterAnchorEl}
				onClose={() => setFilterAnchorEl(null)}
				initial={filters}
				onApply={onChange}
				siteOptions={siteOptions}
				userOptions={userOptions}
			/>

			<Chips filters={filters} onChange={onChange} siteOptions={siteOptions} userOptions={userOptions} />
		</>
	)
}
