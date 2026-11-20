import { Fragment, useCallback, useMemo, useState, type FC } from 'react'
import { Box, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from '@mui/material'

import type { ICategory, ICategoryModifier } from '../../types/category'
import type { ICategoryGroup } from '../../types/categoryGroup'
import type { IGroup } from '@/features/groups/types/group'
import type { CategoryFilterState } from '../CategoryFilters'
import type { CategoryDialogRequest } from '../../types/dialogs'
import { thSx } from './tableStyles'
import { buildBlocks, buildModifiersMap, flatten, moveBlock, moveCategory } from '../../utils/blocks'
import { computeVisibility, isFilteringActive } from '../../utils/tableVisibility'
import { useCategoryOrder } from '../../hooks/useCategoryOrder'
import { useCategoryDnd } from '../../hooks/useCategoryDnd'
import { useCollapse } from '../../hooks/useCollapse'
import { useHiddenToggle } from '../../hooks/useHiddenToggle'
import { CategoryTableToolbar } from './CategoryTableToolbar'
import { RowActionsMenu, type MenuState } from './RowActionsMenu'
import { CategoryBranch, SectionRow } from './rows'
import { OrderSaveBar } from './parts'

const NO_MODIFIERS: ICategoryModifier[] = []

export type CategoryTableProps = {
	categories: ICategory[]
	categoryGroups: ICategoryGroup[]
	groups: IGroup[]
	/** Подкатегории отдельным списком — если у категорий нет вложенных modifiers. */
	modifiers?: ICategoryModifier[]
	filters: CategoryFilterState
	search: string
	onDialog: (request: CategoryDialogRequest) => void
}

export const CategoryTable: FC<CategoryTableProps> = ({
	categories,
	categoryGroups,
	groups,
	modifiers,
	filters,
	search,
	onDialog,
}) => {
	const [showHidden, setShowHidden] = useState(true)
	const [menu, setMenu] = useState<MenuState>(null)
	const closeMenu = useCallback(() => setMenu(null), [])

	const { ordered, isDirty, isSaving, commit, reset: resetOrder, save } = useCategoryOrder(categories)
	const { closed, toggle: toggleOpen, closeAll, openAll } = useCollapse()
	const { toggleCategoryHidden, toggleSectionHidden } = useHiddenToggle()

	const groupsMap = useMemo(() => new Map(groups.map(g => [g.id, g.name])), [groups])
	const modifiersByCategory = useMemo(() => buildModifiersMap(categories, modifiers), [categories, modifiers])
	const blocks = useMemo(() => buildBlocks(ordered, categoryGroups), [ordered, categoryGroups])

	const isFiltering = isFilteringActive(filters, search)

	const visibility = useMemo(
		() =>
			computeVisibility({ blocks, categoryGroups, modifiersByCategory, groupsMap, filters, search, showHidden }),
		[blocks, categoryGroups, modifiersByCategory, groupsMap, filters, search, showHidden],
	)

	const {
		dragging,
		dropTarget,
		handlers: dndHandlers,
	} = useCategoryDnd({ blocks, disabled: isFiltering, onCommit: commit })
	const dropPositionFor = (key: string) => (dropTarget?.key === key ? dropTarget.position : null)

	// Развернуть/свернуть всё
	const collapsibleKeys = useMemo(() => {
		const keys: string[] = []
		blocks.forEach(b => {
			if (b.sectionId) keys.push(b.key)
			b.categories.forEach(c => {
				if ((modifiersByCategory.get(c.id) ?? []).length) keys.push(`cat:${c.id}`)
			})
		})
		return keys
	}, [blocks, modifiersByCategory])

	const allOpen = collapsibleKeys.length > 0 && collapsibleKeys.every(k => !closed.has(k))
	const toggleAll = useCallback(
		(expand: boolean) => {
			if (expand) closeAll(collapsibleKeys)
			else openAll()
		},
		[collapsibleKeys, closeAll, openAll],
	)

	// Счётчик — производное значение, без мутаций во время рендера.
	const shownCount = useMemo(() => {
		let count = visibility.emptySections.length
		visibility.blocks.forEach(({ block, catVis }) => {
			if (block.sectionId) {
				count += 1
				if (!isFiltering && closed.has(block.key)) return
			}
			block.categories.forEach(c => {
				if (catVis.get(c.id)) count += 1
			})
		})
		return count
	}, [visibility, closed, isFiltering])

	const moveRow = useCallback(
		(kind: 'section' | 'category', key: string, categoryId: string | undefined, dir: -1 | 1) => {
			if (kind === 'section') {
				const idx = blocks.findIndex(b => b.key === key)
				const to = idx + dir
				if (idx === -1 || to < 0 || to >= blocks.length) return
				commit(flatten(moveBlock(blocks, key, to + (dir > 0 ? 1 : 0))))
			} else if (categoryId) {
				const block = blocks.find(b => b.key === key)
				if (!block) return
				const idx = block.categories.findIndex(c => c.id === categoryId)
				const to = idx + dir
				if (idx === -1 || to < 0 || to >= block.categories.length) return
				commit(flatten(moveCategory(blocks, key, categoryId, to + (dir > 0 ? 1 : 0))))
			}
		},
		[blocks, commit],
	)

	const isEmpty = visibility.blocks.length === 0 && visibility.emptySections.length === 0

	return (
		<Box>
			<CategoryTableToolbar
				shownCount={shownCount}
				totalCount={categories.length + categoryGroups.length}
				showHidden={showHidden}
				onShowHiddenChange={setShowHidden}
				allOpen={allOpen}
				onToggleAll={toggleAll}
				isFiltering={isFiltering}
			/>

			<TableContainer
				component={Paper}
				elevation={0}
				sx={{
					borderRadius: 3,
					border: '1px solid',
					borderColor: 'divider',
					overflow: 'hidden',
					overflowX: 'auto',
				}}
			>
				<Table sx={{ minWidth: 900 }}>
					<TableHead>
						<TableRow>
							<TableCell sx={thSx}>Название</TableCell>
							<TableCell sx={thSx}>Группа-владелец</TableCell>
							<TableCell sx={thSx}>Приоритет</TableCell>
							<TableCell sx={thSx}>Статус</TableCell>
							<TableCell sx={thSx}>Срок</TableCell>
							<TableCell sx={thSx}>Обновлено</TableCell>
							<TableCell sx={{ width: 52 }} />
						</TableRow>
					</TableHead>

					<TableBody>
						{visibility.blocks.map(({ block, catVis, modVis }) => {
							const section = block.section
							const open = isFiltering || !closed.has(block.key)

							const renderCategory = (cat: ICategory, depth: number, hiddenByAncestor: boolean) =>
								catVis.get(cat.id) ? (
									<CategoryBranch
										key={cat.id}
										category={cat}
										blockKey={block.key}
										depth={depth}
										hiddenByAncestor={hiddenByAncestor}
										groupName={groupsMap.get(cat.groupId)}
										mods={modifiersByCategory.get(cat.id) ?? NO_MODIFIERS}
										modVis={modVis}
										open={isFiltering || !closed.has(`cat:${cat.id}`)}
										onDialog={onDialog}
										onMenu={setMenu}
										onToggleOpen={toggleOpen}
										dnd={dndHandlers}
										dragDisabled={isFiltering}
										dropPosition={dropPositionFor(`category:${cat.id}`)}
										isDragSource={dragging?.kind === 'category' && dragging.categoryId === cat.id}
									/>
								) : null

							if (section) {
								return (
									<Fragment key={block.key}>
										<SectionRow
											section={section}
											blockKey={block.key}
											count={block.categories.length}
											open={open}
											onDialog={onDialog}
											onMenu={setMenu}
											onToggleOpen={toggleOpen}
											dnd={dndHandlers}
											dragDisabled={isFiltering}
											dropPosition={dropPositionFor(`block:${block.key}`)}
											isDragSource={dragging?.kind === 'block' && dragging.key === block.key}
										/>
										{open && block.categories.map(cat => renderCategory(cat, 1, section.isHidden))}
									</Fragment>
								)
							}
							// Корневые категории (и категории с отсутствующей группой — рендерим все, не только первую).
							return (
								<Fragment key={block.key}>
									{block.categories.map(cat => renderCategory(cat, 0, false))}
								</Fragment>
							)
						})}

						{visibility.emptySections.map(section => (
							<SectionRow
								key={`empty:${section.id}`}
								empty
								section={section}
								blockKey={`section:${section.id}`}
								count={0}
								open={false}
								onDialog={onDialog}
								onMenu={setMenu}
								onToggleOpen={toggleOpen}
								dnd={dndHandlers}
								dragDisabled={isFiltering}
								dropPosition={null}
								isDragSource={false}
							/>
						))}
					</TableBody>
				</Table>
				{isEmpty && (
					<Box sx={{ py: 5, textAlign: 'center', color: 'text.secondary' }}>
						Ничего не найдено. Измените запрос или сбросьте фильтры.
					</Box>
				)}
			</TableContainer>

			{isDirty && <OrderSaveBar saving={isSaving} onSave={save} onReset={resetOrder} />}

			<RowActionsMenu
				menu={menu}
				moveDisabled={isFiltering}
				onClose={closeMenu}
				onDialog={onDialog}
				onMove={moveRow}
				onToggleCategoryHidden={toggleCategoryHidden}
				onToggleSectionHidden={toggleSectionHidden}
			/>
		</Box>
	)
}
