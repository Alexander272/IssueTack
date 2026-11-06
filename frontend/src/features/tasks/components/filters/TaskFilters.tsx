import { useState, useMemo, type FC, useCallback } from 'react'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetAvailableUsersQuery, useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { useGetAllEquipmentQuery } from '@/features/equipment/equipmentApiSlice'

import type { FilterValues, TaskFiltersProps } from './types'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { Toolbar } from './Toolbar'
import { Popover } from './Popover'
import { Chips } from './Chips'

export type { FilterValues, TaskFiltersProps }

export const TaskFilters: FC<TaskFiltersProps> = ({
	filters,
	onChange,
	onReset,
	hideGrouping = false,
	statusOptions,
	hideFilters = false,
	equipmentLabel,
}) => {
	const canSeeEquipment = useCheckPermissions({ allOf: [PermRules.Equipment.Read] })
	const [filterAnchorEl, setFilterAnchorEl] = useState<HTMLElement | null>(null)

	const { data: sitesData } = useGetAllSitesQuery()
	const { data: usersData } = useGetAvailableUsersQuery()
	const { data: executorsData } = useGetRealmUsersQuery('executors')
	const { data: equipmentData } = useGetAllEquipmentQuery({ limit: 500, excludeKits: true }, { skip: !canSeeEquipment })

	const siteOptions = useMemo(() => (sitesData?.data ?? []).map(s => ({ id: s.id, label: s.name })), [sitesData])

	const equipmentList = useMemo(
		() => (equipmentData?.data ?? []).map(e => ({ id: e.id, label: e.name })),
		[equipmentData],
	)

	const formatUser = (u: { id: string; lastName: string; firstName: string; username: string }) => ({
		id: u.id,
		label: `${u.lastName} ${u.firstName} (${u.username})`,
	})

	const ownerOptions = useMemo(
		() => (usersData?.data ?? []).filter(u => u.source !== 'mattermost' && !u.isSystem).map(formatUser),
		[usersData],
	)

	const assigneeOptions = useMemo(
		() => (executorsData?.data ?? []).filter(u => u.source !== 'mattermost' && !u.isSystem).map(formatUser),
		[executorsData],
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
		if (filters.equipmentId) count++
		return count
	}, [filters])

	const handleOpenFilter = useCallback((e: React.MouseEvent<HTMLElement>) => {
		setFilterAnchorEl(e.currentTarget)
	}, [])

	const handleCloseFilter = useCallback(() => {
		setFilterAnchorEl(null)
	}, [])

	return (
		<>
			<Toolbar
				search={filters.search}
				groupEnabled={filters.groupEnabled}
				groupBy={filters.groupBy}
				activeCount={activeCount}
				hideGrouping={hideGrouping}
				onSearchChange={search => onChange({ search })}
				onGroupChange={option => onChange({ groupBy: option ?? 'category', groupEnabled: Boolean(option) })}
				onOpenFilter={handleOpenFilter}
				onReset={onReset}
				hideFilters={hideFilters}
			/>

			{!hideFilters && (
				<Popover
					open={Boolean(filterAnchorEl)}
					anchorEl={filterAnchorEl}
					onClose={handleCloseFilter}
					filters={filters}
					onApply={onChange}
					siteOptions={siteOptions}
					ownerOptions={ownerOptions}
					assigneeOptions={assigneeOptions}
					statusOptions={statusOptions}
					equipmentOptions={equipmentList}
				/>
			)}

			{!hideFilters && (
				<Chips
					filters={filters}
					onChange={onChange}
					siteOptions={siteOptions}
					ownerOptions={ownerOptions}
					assigneeOptions={assigneeOptions}
					equipmentLabel={equipmentLabel}
					equipmentOptions={equipmentList}
				/>
			)}
		</>
	)
}
