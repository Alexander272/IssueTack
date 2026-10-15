import type { FC } from 'react'
import { Box, IconButton, TableCell, TableRow, Tooltip, Typography } from '@mui/material'
import { BoxesIcon, ChevronDownIcon, ChevronRightIcon, MicrochipIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { useEquipmentTree } from '../treeState'
import { EquipmentKindIcon } from '../EquipmentKindIcon'
import { EquipmentStatusBadge } from '../EquipmentStatusBadge'
import { pluralize } from '@/utils/plural'

// Узел дерева дополняет единицу серверными признаками раскрытия; плоский список
// приходит обычной единицей, поэтому они опциональны.
type RowNode = IEquipment & { hasChildren?: boolean; childCount?: number }

type Props = {
	node: RowNode
	depth?: number
}

// Строка дерева оборудования: иконка типа (с бейджем вложенности), название с
// подписью и колонки типа/держателя/серийника/статуса. Раскрытие и открытие берёт
// из контекста, поэтому снаружи приходят только узел и глубина.
export const EquipmentRow: FC<Props> = ({ node, depth = 0 }) => {
	const { tree, expanded, toggle, open } = useEquipmentTree()
	const accent = equipmentAccent(node)
	const childCount = node.childCount ?? 0
	const holder = node.holderFullName || node.holderName
	const location = node.location || node.siteName
	const isInstalled = Boolean(node.parentId)
	const hasToggle = tree && (Boolean(node.hasChildren) || childCount > 0)
	const isExpanded = hasToggle && expanded.has(node.id)
	const nesting = childCount > 0 && !node.isKit

	const rowBg = node.isKit ? '#7c3aed08' : isInstalled ? '#ea580c08' : depth > 0 ? '#fafafa' : undefined
	const hoverBg = node.isKit ? '#f5f3ff' : isInstalled ? '#fff7ed' : '#f5f5f5'

	const second = [`Инв. №${node.inventoryNumber ?? '—'}`]
	if (isInstalled) second.push(`внутри ${node.containerName ?? 'родителя'}`)
	else if (childCount) second.push(`${childCount} ${pluralize(childCount, ['компонент', 'компонента', 'компонентов'])}`)
	else if (node.status === 'in_stock' && location) second.push(location)

	return (
		<TableRow
			hover
			onClick={() => open(node.id)}
			sx={{
				cursor: 'pointer',
				bgcolor: rowBg,
				borderBottom: '1px solid #f3f4f6',
				'&:hover': { bgcolor: hoverBg },
				'&:last-child td': { border: 0 },
			}}
		>
			<TableCell>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0, pl: depth * 2.5 }}>
					{depth > 0 && (
						<Box
							component='span'
							sx={{
								color: '#d1d5db',
								fontFamily: 'monospace',
								fontSize: 14,
								lineHeight: 1,
								flexShrink: 0,
							}}
						>
							└
						</Box>
					)}
					{hasToggle ? (
						<IconButton
							size='small'
							aria-label={isExpanded ? 'Свернуть' : 'Развернуть'}
							onClick={e => {
								e.stopPropagation()
								toggle(node.id)
							}}
							sx={{ color: '#9ca3af', p: 0.25, flexShrink: 0 }}
						>
							<ChevronDownIcon
								sx={{
									fontSize: 16,
									transform: isExpanded ? 'rotate(0deg)' : 'rotate(-90deg)',
									transition: 'transform 0.15s ease',
								}}
							/>
						</IconButton>
					) : (
						<Box sx={{ width: 24, flexShrink: 0 }} />
					)}

					<Box sx={{ position: 'relative', flexShrink: 0 }}>
						<Box
							sx={{
								width: 34,
								height: 34,
								borderRadius: '10px',
								bgcolor: `${accent}1a`,
								color: accent,
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'center',
							}}
						>
							<EquipmentKindIcon isKit={node.isKit} kind={node.typeKind} sx={{ fontSize: 18 }} />
						</Box>
						{nesting && (
							<Box
								sx={{
									position: 'absolute',
									right: -3,
									bottom: -3,
									width: 16,
									height: 16,
									borderRadius: '50%',
									bgcolor: '#e5e7eb',
									border: '2px solid #fff',
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'center',
								}}
							>
								<MicrochipIcon sx={{ fontSize: 9, color: '#4b5563' }} />
							</Box>
						)}
					</Box>

					<Box sx={{ minWidth: 0, flexGrow: 1 }}>
						<Box sx={{ display: 'flex', alignItems: 'baseline', gap: 0.5, minWidth: 0 }}>
							<Tooltip title={node.name} placement='top'>
								<Typography
									sx={{
										fontWeight: 600,
										color: '#111827',
										fontSize: '0.875rem',
										overflow: 'hidden',
										textOverflow: 'ellipsis',
										whiteSpace: 'nowrap',
									}}
								>
									{node.name}
								</Typography>
							</Tooltip>
							{hasToggle && !isExpanded && (
								<Typography
									component='span'
									sx={{ fontSize: '0.75rem', color: '#9ca3af', fontWeight: 500, flexShrink: 0 }}
								>
									({childCount})
								</Typography>
							)}
						</Box>
						<Typography
							sx={{
								fontSize: '0.75rem',
								color: '#6b7280',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap',
							}}
						>
							{second.join(' · ')}
						</Typography>
					</Box>
				</Box>
			</TableCell>

			<TableCell>
				{node.isKit ? (
					<Box
						component='span'
						sx={{
							display: 'inline-flex',
							alignItems: 'center',
							gap: 0.5,
							px: 1.25,
							py: 0.25,
							borderRadius: '999px',
							fontSize: '0.7rem',
							fontWeight: 600,
							bgcolor: '#f3e8ff',
							color: '#7c3aed',
						}}
					>
						<BoxesIcon sx={{ fontSize: 13 }} />
						Комплект
					</Box>
				) : (
					<Typography sx={{ fontSize: '0.875rem', color: '#6b7280' }} noWrap>
						{node.typeName}
					</Typography>
				)}
			</TableCell>

			<TableCell>
				<Box sx={{ minWidth: 0 }}>
					<Typography
						sx={{ fontSize: '0.8125rem', fontWeight: 500, color: holder ? '#374151' : '#9ca3af' }}
						noWrap
					>
						{holder || '—'}
					</Typography>
					{location && (
						<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af' }} noWrap>
							{location}
						</Typography>
					)}
				</Box>
			</TableCell>

			<TableCell>
				<Typography
					sx={{
						fontFamily: 'monospace',
						fontSize: '0.75rem',
						color: node.serialNumber ? '#4b5563' : '#9ca3af',
					}}
					noWrap
				>
					{node.serialNumber ?? '—'}
				</Typography>
			</TableCell>

			<TableCell>
				<EquipmentStatusBadge status={node.status} />
			</TableCell>

			<TableCell align='right'>
				<ChevronRightIcon sx={{ fontSize: 18, color: '#cbd5e1' }} />
			</TableCell>
		</TableRow>
	)
}
