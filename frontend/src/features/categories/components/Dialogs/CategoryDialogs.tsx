import { type FC } from 'react'

import type { IGroup } from '@/features/groups/types/group'
import type { ICategory } from '../../types/category'
import type { ICategoryGroup } from '../../types/categoryGroup'
import type { CategoryDialogState } from '../../types/dialogs'
import { CategoryDialog } from './CategoryDialog'
import { CategoryModifierDialog } from './CategoryModifierDialog'
import { CategoryGroupDialog } from './CategoryGroupDialog'

type Props = {
	state: CategoryDialogState
	groups: IGroup[]
	categories: ICategory[]
	categoryGroups: ICategoryGroup[]
	onClose: () => void
}

export const CategoryDialogs: FC<Props> = ({ state, groups, categories, categoryGroups, onClose }) => (
	<>
		<CategoryDialog
			open={state?.kind === 'category'}
			category={state?.kind === 'category' ? state.data : undefined}
			groups={groups}
			categoryGroups={categoryGroups}
			onClose={onClose}
		/>
		<CategoryGroupDialog
			open={state?.kind === 'section'}
			categoryGroup={state?.kind === 'section' ? state.data : undefined}
			onClose={onClose}
		/>
		<CategoryModifierDialog
			open={state?.kind === 'modifier'}
			modifier={state?.kind === 'modifier' ? state.data : undefined}
			categories={categories}
			onClose={onClose}
		/>
	</>
)
