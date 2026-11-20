import { memo } from 'react'
import { Box, MenuItem, MenuList, Popover } from '@mui/material'
import { ChevronDownIcon, ChevronUpIcon, EditIcon, EyeIcon, EyeOffIcon, PlusIcon } from 'lucide-mui'

import type { ICategory, ICategoryModifier } from '../../types/category'
import type { ICategoryGroup } from '../../types/categoryGroup'
import type { CategoryDialogRequest } from '../../types/dialogs'

export type MenuState =
	| { anchor: HTMLElement; kind: 'section'; section: ICategoryGroup; empty?: boolean }
	| { anchor: HTMLElement; kind: 'category'; category: ICategory; blockKey: string }
	| { anchor: HTMLElement; kind: 'modifier'; modifier: ICategoryModifier }
	| null

const iconSx = { fontSize: 16, mr: 1 }
const MenuDivider = () => <Box sx={{ borderTop: '1px solid', borderColor: 'divider', my: 0.5 }} />

type Props = {
	menu: MenuState
	moveDisabled: boolean
	onClose: () => void
	onDialog: (request: CategoryDialogRequest) => void
	onMove: (kind: 'section' | 'category', key: string, categoryId: string | undefined, dir: -1 | 1) => void
	onToggleCategoryHidden: (category: ICategory) => void
	onToggleSectionHidden: (section: ICategoryGroup) => void
}

export const RowActionsMenu = memo(function RowActionsMenu({
	menu,
	moveDisabled,
	onClose,
	onDialog,
	onMove,
	onToggleCategoryHidden,
	onToggleSectionHidden,
}: Props) {
	const run = (action: () => void) => () => {
		onClose()
		action()
	}

	return (
		<Popover
			open={Boolean(menu)}
			anchorEl={menu?.anchor ?? null}
			onClose={onClose}
			anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
			transformOrigin={{ vertical: 'top', horizontal: 'right' }}
			slotProps={{ paper: { sx: { borderRadius: 2.5, minWidth: 220, p: 0.5 } } }}
		>
			{menu?.kind === 'section' && (
				<MenuList>
					<MenuItem onClick={run(() => onDialog({ kind: 'create-category', sectionId: menu.section.id }))}>
						<PlusIcon sx={iconSx} /> Добавить категорию
					</MenuItem>
					<MenuItem onClick={run(() => onDialog({ kind: 'edit-section', section: menu.section }))}>
						<EditIcon sx={iconSx} /> Редактировать
					</MenuItem>
					<MenuItem onClick={run(() => onToggleSectionHidden(menu.section))}>
						{menu.section.isHidden ? <EyeIcon sx={iconSx} /> : <EyeOffIcon sx={iconSx} />}
						{menu.section.isHidden ? 'Показать в списках' : 'Скрыть из списков'}
					</MenuItem>
					<MenuDivider />
					{/* Пустые разделы не участвуют в порядке — пункты заблокированы. */}
					<MenuItem
						disabled={moveDisabled || menu.empty}
						onClick={run(() => onMove('section', `section:${menu.section.id}`, undefined, -1))}
					>
						<ChevronUpIcon sx={iconSx} /> Выше
					</MenuItem>
					<MenuItem
						disabled={moveDisabled || menu.empty}
						onClick={run(() => onMove('section', `section:${menu.section.id}`, undefined, 1))}
					>
						<ChevronDownIcon sx={iconSx} /> Ниже
					</MenuItem>
				</MenuList>
			)}
			{menu?.kind === 'category' && (
				<MenuList>
					<MenuItem onClick={run(() => onDialog({ kind: 'create-modifier', categoryId: menu.category.id }))}>
						<PlusIcon sx={iconSx} /> Добавить подкатегорию
					</MenuItem>
					<MenuItem onClick={run(() => onDialog({ kind: 'edit-category', category: menu.category }))}>
						<EditIcon sx={iconSx} /> Редактировать
					</MenuItem>
					<MenuItem onClick={run(() => onToggleCategoryHidden(menu.category))}>
						{menu.category.isHidden ? <EyeIcon sx={iconSx} /> : <EyeOffIcon sx={iconSx} />}
						{menu.category.isHidden ? 'Показать в списках' : 'Скрыть из списков'}
					</MenuItem>
					<MenuDivider />
					<MenuItem
						disabled={moveDisabled}
						onClick={run(() => onMove('category', menu.blockKey, menu.category.id, -1))}
					>
						<ChevronUpIcon sx={iconSx} /> Выше
					</MenuItem>
					<MenuItem
						disabled={moveDisabled}
						onClick={run(() => onMove('category', menu.blockKey, menu.category.id, 1))}
					>
						<ChevronDownIcon sx={iconSx} /> Ниже
					</MenuItem>
				</MenuList>
			)}
			{menu?.kind === 'modifier' && (
				<MenuList>
					<MenuItem onClick={run(() => onDialog({ kind: 'edit-modifier', modifier: menu.modifier }))}>
						<EditIcon sx={iconSx} /> Редактировать
					</MenuItem>
				</MenuList>
			)}
		</Popover>
	)
})
