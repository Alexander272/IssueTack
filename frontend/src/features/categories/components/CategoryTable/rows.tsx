import { memo, type DragEvent } from 'react'
import { Box, Stack, TableCell, TableRow, Typography } from '@mui/material'
import { CornerDownRightIcon, FolderIcon, LayersIcon, TagIcon } from 'lucide-mui'

import type { ICategory, ICategoryModifier } from '../../types/category'
import type { ICategoryGroup } from '../../types/categoryGroup'
import type { CategoryDndHandlers } from '../../hooks/useCategoryDnd'
import type { DropPosition } from '../../utils/blocks'
import type { CategoryDialogRequest } from '../../types/dialogs'
import type { MenuState } from './RowActionsMenu'
import { ACCENT, dropBordersSx, rowCellSx, statusLabel } from './tableStyles'
import { categoryHiddenLabel, isCategoryHidden } from '../../utils/visibility'
import { TaskPriorityBadge } from '@/features/tasks/components/TaskPriorityBadge'
import { StatusBadge } from '@/features/access/components/StatusBadge'
import {
	CountBadge,
	DueDaysCell,
	ExpandButton,
	HiddenMarker,
	NameCell,
	RowGrip,
	RowMenuCell,
	UpdatedAtCell,
} from './parts'

type CommonRowProps = {
	onDialog: (request: CategoryDialogRequest) => void
	onMenu: (menu: MenuState) => void
	onToggleOpen: (key: string) => void
}

type DndRowProps = {
	dnd: CategoryDndHandlers
	dragDisabled: boolean
	dropPosition: DropPosition | null
	isDragSource: boolean
}

/* ---------- Раздел (в т.ч. пустой) ---------- */

type SectionRowProps = CommonRowProps &
	DndRowProps & {
		section: ICategoryGroup
		blockKey: string
		count: number
		open: boolean
		empty?: boolean
	}

export const SectionRow = memo(function SectionRow({
	section,
	blockKey,
	count,
	open,
	empty,
	onDialog,
	onMenu,
	onToggleOpen,
	dnd,
	dragDisabled,
	dropPosition,
	isDragSource,
}: SectionRowProps) {
	const hidden = section.isHidden
	const target = { kind: 'block' as const, key: blockKey }
	const dndProps = empty
		? {}
		: {
				draggable: !dragDisabled,
				onDragStart: (e: DragEvent<HTMLElement>) => dnd.start(e, target, e.currentTarget),
				onDragEnd: dnd.reset,
				onDragOver: (e: DragEvent<HTMLElement>) => dnd.over(e, target),
				onDragLeave: dnd.leave,
				onDrop: (e: DragEvent<HTMLElement>) => dnd.drop(e, target),
			}

	return (
		<TableRow
			hover
			{...dndProps}
			onClick={() => onDialog({ kind: 'edit-section', section })}
			sx={{
				cursor: 'pointer',
				opacity: isDragSource ? 0.4 : hidden ? 0.55 : 1,
				'&:hover .row-grip': { opacity: 1 },
				'& td': {
					bgcolor: empty ? '#f9fafb' : '#f3f4fb',
					fontWeight: 600,
					...rowCellSx,
					...(empty ? {} : dropBordersSx(dropPosition)),
				},
				'&:hover td': { bgcolor: empty ? '#f3f4f6' : '#e9ebf7' },
			}}
		>
			<TableCell sx={{ pl: 1.75 }}>
				<Stack direction='row' spacing={0.75} sx={{ alignItems: 'center' }}>
					{empty ? <Box sx={{ width: 16 }} /> : <RowGrip enabled={!dragDisabled} />}
					{empty ? (
						<Box sx={{ width: 22 }} />
					) : (
						<ExpandButton open={open} onToggle={() => onToggleOpen(blockKey)} />
					)}
					<LayersIcon sx={{ fontSize: 16, color: empty ? 'text.secondary' : ACCENT }} />
					<NameCell
						name={section.name}
						weight={600}
						italic={hidden}
						adornment={
							<>
								<CountBadge>{count} кат.</CountBadge>
								{hidden && <HiddenMarker label='Скрыт из списков выбора' />}
							</>
						}
					/>
				</Stack>
			</TableCell>
			<TableCell />
			<TableCell />
			<TableCell />
			<TableCell />
			<UpdatedAtCell value={section.updatedAt} fontWeight={400} />
			<RowMenuCell onOpen={anchor => onMenu({ anchor, kind: 'section', section, empty })} />
		</TableRow>
	)
})

/* ---------- Категория ---------- */

type CategoryRowProps = CommonRowProps &
	DndRowProps & {
		category: ICategory
		blockKey: string
		depth: number
		hiddenByAncestor: boolean
		groupName?: string
		modifiersCount: number
		open: boolean
	}

export const CategoryRow = memo(function CategoryRow({
	category,
	blockKey,
	depth,
	hiddenByAncestor,
	groupName,
	modifiersCount,
	open,
	onDialog,
	onMenu,
	onToggleOpen,
	dnd,
	dragDisabled,
	dropPosition,
	isDragSource,
}: CategoryRowProps) {
	const effHidden = isCategoryHidden(category) || hiddenByAncestor
	const hasKids = modifiersCount > 0
	const payload = { kind: 'category' as const, key: blockKey, categoryId: category.id }

	return (
		<TableRow
			hover
			draggable={!dragDisabled}
			onDragStart={e => dnd.start(e, payload, e.currentTarget)}
			onDragEnd={dnd.reset}
			onDragOver={e => dnd.over(e, payload)}
			onDragLeave={dnd.leave}
			onDrop={e => dnd.drop(e, payload)}
			onClick={() => onDialog({ kind: 'edit-category', category })}
			sx={{
				cursor: 'pointer',
				opacity: isDragSource ? 0.4 : effHidden ? 0.55 : category.isActive ? 1 : 0.75,
				'&:hover .row-grip': { opacity: 1 },
				'& td': { ...rowCellSx, ...dropBordersSx(dropPosition) },
			}}
		>
			<TableCell sx={{ pl: 1.75 + depth * 3.5 }}>
				<Stack direction='row' spacing={0.75} sx={{ alignItems: 'center' }}>
					<RowGrip enabled={!dragDisabled} />
					{hasKids ? (
						<ExpandButton open={open} onToggle={() => onToggleOpen(`cat:${category.id}`)} />
					) : (
						<Box sx={{ width: 22 }} />
					)}
					{depth === 0 ? (
						<TagIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
					) : (
						<FolderIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
					)}
					<NameCell
						name={category.name}
						description={category.description}
						italic={effHidden}
						adornment={
							<>
								{hasKids && <CountBadge>{modifiersCount} подкат.</CountBadge>}
								{effHidden && <HiddenMarker label={categoryHiddenLabel(category)} />}
							</>
						}
					/>
				</Stack>
			</TableCell>
			<TableCell>
				<Typography
					component='span'
					sx={{
						display: 'inline-block',
						px: 1.125,
						py: 0.25,
						borderRadius: '6px',
						bgcolor: '#eceefc',
						color: ACCENT,
						fontSize: '0.78rem',
					}}
				>
					{groupName || '—'}
				</Typography>
			</TableCell>
			<TableCell>
				<TaskPriorityBadge priority={category.priority} />
			</TableCell>
			<TableCell>
				<StatusBadge active={category.isActive} label={statusLabel(category.isActive)} />
			</TableCell>
			<DueDaysCell days={category.defaultDueDays} />
			<UpdatedAtCell value={category.updatedAt} />
			<RowMenuCell onOpen={anchor => onMenu({ anchor, kind: 'category', category, blockKey })} />
		</TableRow>
	)
})

/* ---------- Подкатегория ---------- */

type ModifierRowProps = {
	modifier: ICategoryModifier
	fallbackPriority: ICategory['priority']
	depth: number
	onDialog: (request: CategoryDialogRequest) => void
	onMenu: (menu: MenuState) => void
}

export const ModifierRow = memo(function ModifierRow({
	modifier,
	fallbackPriority,
	depth,
	onDialog,
	onMenu,
}: ModifierRowProps) {
	return (
		<TableRow
			hover
			onClick={() => onDialog({ kind: 'edit-modifier', modifier })}
			sx={{ cursor: 'pointer', opacity: modifier.isActive ? 1 : 0.55, '& td': rowCellSx }}
		>
			<TableCell sx={{ pl: 1.75 + depth * 3.5 }}>
				<Stack direction='row' spacing={0.75} sx={{ alignItems: 'center' }}>
					<Box sx={{ width: 16 }} />
					<Box sx={{ width: 22 }} />
					<CornerDownRightIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
					<NameCell
						name={modifier.name}
						description={modifier.description}
						weight={400}
						italic={!modifier.isActive}
					/>
				</Stack>
			</TableCell>
			<TableCell />
			<TableCell>
				<TaskPriorityBadge priority={modifier.priority || fallbackPriority} />
			</TableCell>
			<TableCell>
				<StatusBadge active={modifier.isActive} label={statusLabel(modifier.isActive)} />
			</TableCell>
			<DueDaysCell days={modifier.defaultDueDays} />
			<UpdatedAtCell value={modifier.updatedAt} />
			<RowMenuCell onOpen={anchor => onMenu({ anchor, kind: 'modifier', modifier })} />
		</TableRow>
	)
})

/* ---------- Категория + её видимые подкатегории ---------- */

type CategoryBranchProps = CommonRowProps &
	DndRowProps & {
		category: ICategory
		blockKey: string
		depth: number
		hiddenByAncestor: boolean
		groupName?: string
		mods: ICategoryModifier[]
		modVis: ReadonlyMap<string, boolean>
		open: boolean
	}

export const CategoryBranch = memo(function CategoryBranch({ mods, modVis, open, ...rowProps }: CategoryBranchProps) {
	return (
		<>
			<CategoryRow {...rowProps} modifiersCount={mods.length} open={open} />
			{open &&
				mods.map(m =>
					modVis.get(m.id) ? (
						<ModifierRow
							key={m.id}
							modifier={m}
							fallbackPriority={rowProps.category.priority}
							depth={rowProps.depth + 1}
							onDialog={rowProps.onDialog}
							onMenu={rowProps.onMenu}
						/>
					) : null,
				)}
		</>
	)
})
