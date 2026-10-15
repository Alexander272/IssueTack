import { Fragment, type FC, useMemo } from 'react'
import { TableCell, TableRow } from '@mui/material'

import type { IEquipmentTreeNode } from '../../types/equipment'
import { Loader } from '@/components/Fallback/Loader'
import { useGetEquipmentTreeQuery } from '../../equipmentApiSlice'
import { useEquipmentTree } from '../treeState'
import { EQUIPMENT_TABLE_COLUMNS, EquipmentGroupRow, EquipmentRow } from '../Table'
import { sortByInventory, sortNodes, type RootBundle } from './rows'

type RootRowsProps = {
	bundles: RootBundle[]
}

// Корневые строки страницы: «партии» складских единиц и одиночные узлы дерева.
// Раскрытие и открытие карточки строки берут из контекста — пропсами не протаскиваются.
export const EquipmentRootRows: FC<RootRowsProps> = ({ bundles }) => {
	const { expanded, toggle } = useEquipmentTree()

	return (
		<>
			{bundles.map(bundle =>
				bundle.type === 'group' ? (
					<Fragment key={`group-${bundle.key}`}>
						<EquipmentGroupRow
							items={bundle.items}
							expanded={expanded.has(bundle.key)}
							onToggle={() => toggle(bundle.key)}
						/>
						{expanded.has(bundle.key) &&
							sortByInventory(bundle.items).map(node => <EquipmentRow key={node.id} node={node} />)}
					</Fragment>
				) : (
					<TreeNode key={bundle.node.id} node={bundle.node} depth={0} />
				),
			)}
		</>
	)
}

type NodeProps = {
	node: IEquipmentTreeNode
	depth: number
}

// Узел дерева со своими детьми: дети грузятся отдельным запросом при раскрытии.
const TreeNode: FC<NodeProps> = ({ node, depth }) => {
	const { expanded } = useEquipmentTree()

	return (
		<Fragment>
			<EquipmentRow node={node} depth={depth} />
			{expanded.has(node.id) && node.hasChildren && <NodeChildren parentId={node.id} depth={depth + 1} />}
		</Fragment>
	)
}

type ChildrenProps = {
	parentId: string
	depth: number
}

// Дети узла: отдельный (ленивый) уровень дерева со своим RTK-запросом, поэтому
// инвалидация тега Equipment перерисовывает уже раскрытые ветки.
const NodeChildren: FC<ChildrenProps> = ({ parentId, depth }) => {
	const { data, isFetching } = useGetEquipmentTreeQuery({ parentId })
	const nodes = useMemo(() => sortNodes(data?.data ?? []), [data])

	if (isFetching && nodes.length === 0) {
		return (
			<TableRow>
				<TableCell colSpan={EQUIPMENT_TABLE_COLUMNS} align='center' sx={{ py: 2 }}>
					<Loader />
				</TableCell>
			</TableRow>
		)
	}

	return (
		<>
			{nodes.map(node => (
				<TreeNode key={node.id} node={node} depth={depth} />
			))}
		</>
	)
}
