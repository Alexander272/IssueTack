import { Fragment, type FC, useMemo } from 'react'
import { TableCell, TableRow } from '@mui/material'

import type { IEquipmentTreeNode } from '../../types/equipment'
import { Loader } from '@/components/Fallback/Loader'
import { useGetEquipmentTreeQuery } from '../../equipmentApiSlice'
import { EQUIPMENT_TABLE_COLUMNS, EquipmentGroupRow, EquipmentRow } from '../Table'
import { sortByInventory, sortNodes, type RootBundle } from './rows'

type TreeProps = {
	expanded: Set<string>
	onToggle: (id: string) => void
	onOpen: (id: string) => void
}

type RootRowsProps = TreeProps & {
	bundles: RootBundle[]
}

// Корневые строки страницы: «партии» складских единиц и одиночные узлы дерева.
export const EquipmentRootRows: FC<RootRowsProps> = ({ bundles, expanded, onToggle, onOpen }) => (
	<>
		{bundles.map(bundle =>
			bundle.type === 'group' ? (
				<Fragment key={`group-${bundle.key}`}>
					<EquipmentGroupRow
						items={bundle.items}
						expanded={expanded.has(bundle.key)}
						columnCount={EQUIPMENT_TABLE_COLUMNS}
						onToggle={() => onToggle(bundle.key)}
					/>
					{expanded.has(bundle.key) &&
						sortByInventory(bundle.items).map(node => (
							<EquipmentRow key={node.id} node={node} depth={0} showToggle={false} onOpen={onOpen} />
						))}
				</Fragment>
			) : (
				<TreeNode
					key={bundle.node.id}
					node={bundle.node}
					depth={0}
					expanded={expanded}
					onToggle={onToggle}
					onOpen={onOpen}
				/>
			),
		)}
	</>
)

type NodeProps = TreeProps & {
	node: IEquipmentTreeNode
	depth: number
	parentName?: string
}

// Узел дерева со своими детьми: дети грузятся отдельным запросом при раскрытии.
const TreeNode: FC<NodeProps> = ({ node, depth, parentName, expanded, onToggle, onOpen }) => (
	<Fragment>
		<EquipmentRow
			node={node}
			depth={depth}
			parentName={parentName}
			showToggle
			expanded={expanded.has(node.id)}
			onToggle={() => onToggle(node.id)}
			onOpen={onOpen}
		/>
		{expanded.has(node.id) && node.hasChildren && (
			<NodeChildren
				parentId={node.id}
				parentName={node.name}
				depth={depth + 1}
				expanded={expanded}
				onToggle={onToggle}
				onOpen={onOpen}
			/>
		)}
	</Fragment>
)

type ChildrenProps = TreeProps & {
	parentId: string
	parentName: string
	depth: number
}

// Дети узла: отдельный (ленивый) уровень дерева со своим RTK-запросом, поэтому
// инвалидация тега Equipment перерисовывает уже раскрытые ветки.
const NodeChildren: FC<ChildrenProps> = ({ parentId, parentName, depth, expanded, onToggle, onOpen }) => {
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
				<TreeNode
					key={node.id}
					node={node}
					depth={depth}
					parentName={parentName}
					expanded={expanded}
					onToggle={onToggle}
					onOpen={onOpen}
				/>
			))}
		</>
	)
}
