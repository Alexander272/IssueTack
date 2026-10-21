import { Fragment, type FC, useMemo } from 'react'
import { Box, TableCell, TableRow } from '@mui/material'

import type { IEquipmentTreeNode } from '../../types/equipment'
import { Loader } from '@/components/Fallback/Loader'
import { useGetEquipmentTreeQuery } from '../../equipmentApiSlice'
import { useEquipmentTree } from '../treeState'
import { EQUIPMENT_TABLE_COLUMNS, EquipmentCard, EquipmentGroupRow, EquipmentRow } from '../Table'
import { sortByInventory, sortNodes, type RootBundle } from './rows'

type RootRowsProps = {
	bundles: RootBundle[]
}

// Корневые строки страницы: «партии» складских единиц и одиночные узлы дерева.
// Раскрытие и открытие карточки строки берут из контекста — пропсами не протаскиваются.
export const EquipmentRootRows: FC<RootRowsProps> = ({ bundles }) => {
	const { expanded, toggle, mobile } = useEquipmentTree()

	// Мобильный вариант: корневая единица — карточка-контейнер, дети — компактные
	// строки внутри неё (см. MobileTreeNode). «Партия» — тоже контейнер, её позиции
	// лежат внутри её карточки, а не отдельными белыми блоками.
	if (mobile) {
		return (
			<>
				{bundles.map(bundle =>
					bundle.type === 'group' ? (
						<EquipmentGroupRow
							key={`group-${bundle.key}`}
							items={bundle.items}
							expanded={expanded.has(bundle.key)}
							onToggle={() => toggle(bundle.key)}
						>
							{sortByInventory(bundle.items).map(node => (
								<MobileTreeNode key={node.id} node={node} embedded />
							))}
						</EquipmentGroupRow>
					) : (
						<MobileTreeNode key={bundle.node.id} node={bundle.node} />
					),
				)}
			</>
		)
	}

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

type MobileNodeProps = {
	node: IEquipmentTreeNode
	embedded?: boolean
}

// Мобильный узел дерева: карточка (корневая или вложенная) плюс раскрытые потомки
// внутри неё. Отступ и линию-связку вложенного списка задаёт сама EquipmentCard
// (nestedListSx), поэтому глубина пропсом не передаётся.
const MobileTreeNode: FC<MobileNodeProps> = ({ node, embedded }) => {
	const { expanded } = useEquipmentTree()

	return (
		<EquipmentCard node={node} embedded={embedded}>
			{expanded.has(node.id) && node.hasChildren && <NodeChildren parentId={node.id} depth={0} />}
		</EquipmentCard>
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
	const { mobile, search } = useEquipmentTree()
	const { data, isFetching } = useGetEquipmentTreeQuery({ parentId, search: search || undefined })
	const nodes = useMemo(() => sortNodes(data?.data ?? []), [data])

	if (isFetching && nodes.length === 0) {
		if (mobile) {
			return (
				<Box sx={{ py: 2, display: 'flex', justifyContent: 'center' }}>
					<Loader />
				</Box>
			)
		}
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
			{nodes.map(node =>
				mobile ? (
					<MobileTreeNode key={node.id} node={node} embedded />
				) : (
					<TreeNode key={node.id} node={node} depth={depth} />
				),
			)}
		</>
	)
}
