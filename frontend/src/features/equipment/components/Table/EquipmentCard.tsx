import type { FC, ReactNode } from 'react'
import { Box, IconButton, Stack, Tooltip, Typography } from '@mui/material'
import { BoxesIcon, ChevronDownIcon, MicrochipIcon } from 'lucide-mui'

import { equipmentAccent } from '../../utils/typeKind'
import { useEquipmentTree } from '../treeState'
import { EquipmentKindIcon } from '../EquipmentKindIcon'
import { EquipmentStatusBadge } from '../EquipmentStatusBadge'
import { equipmentRowMeta, type EquipmentRowNode } from './rowMeta'
import { nestedListSx } from './nestedList'

type Props = {
	node: EquipmentRowNode
	depth?: number
	// embedded — строка внутри карточки-родителя (вложенный элемент дерева): без
	// собственной рамки/фона/тени, компактнее. Карточку-контейнер рисует TreeNode.
	embedded?: boolean
	// children — раскрытые потомки узла: рисуются внутри этой же карточки списком
	// с линией-связкой, а не отдельными белыми карточками.
	children?: ReactNode
}

// Мобильная карточка единицы — аналог строки EquipmentRow на узком экране (как
// TaskCard против строки таблицы заявок). Иерархия сохраняется: узел с детьми рисует
// шеврон и счётчик «(N)», а раскрытые потомки выводятся внутри родительской карточки
// компактным списком (embedded). Статус вынесен в нижнюю строку — название получает
// всю ширину карточки и переносится максимум на две строки.
export const EquipmentCard: FC<Props> = ({ node, embedded = false, children }) => {
	const { tree, expanded, toggle, open } = useEquipmentTree()
	const accent = equipmentAccent(node)
	const { holder, subtitle, childCount } = equipmentRowMeta(node)
	const hasToggle = tree && (Boolean(node.hasChildren) || childCount > 0)
	const isExpanded = hasToggle && expanded.has(node.id)
	const nesting = childCount > 0 && !node.isKit

	const iconSize = embedded ? 30 : 36

	return (
		<Box
			onClick={() => open(node.id)}
			sx={
				embedded
					? {
							p: 0.75,
							borderRadius: 1.5,
							cursor: 'pointer',
							transition: 'background-color 0.15s ease',
							'&:hover': { bgcolor: 'rgba(15, 23, 42, 0.04)' },
						}
					: {
							bgcolor: node.isKit ? '#7c3aed0a' : node.parentId ? '#ea580c0a' : '#fff',
							border: '1px solid #e5e7eb',
							borderRadius: 2.5,
							p: 1.5,
							cursor: 'pointer',
							overflow: 'hidden',
							transition: 'box-shadow 0.2s ease, border-color 0.2s ease',
							'&:hover': { boxShadow: '0 2px 8px rgba(0,0,0,0.06)', borderColor: '#d1d5db' },
						}
			}
		>
			<Stack direction='row' spacing={1} sx={{ alignItems: 'flex-start', minWidth: 0 }}>
				{hasToggle ? (
					<IconButton
						size='small'
						aria-label={isExpanded ? 'Свернуть' : 'Развернуть'}
						onClick={e => {
							e.stopPropagation()
							toggle(node.id)
						}}
						sx={{ color: '#9ca3af', p: 0.25, mt: 0.25, flexShrink: 0 }}
					>
						<ChevronDownIcon
							sx={{
								fontSize: embedded ? 16 : 18,
								transform: isExpanded ? 'rotate(0deg)' : 'rotate(-90deg)',
								transition: 'transform 0.15s ease',
							}}
						/>
					</IconButton>
				) : (
					<Box sx={{ width: embedded ? 22 : 26, flexShrink: 0 }} />
				)}

				<Box sx={{ position: 'relative', flexShrink: 0 }}>
					<Box
						sx={{
							width: iconSize,
							height: iconSize,
							borderRadius: '10px',
							bgcolor: `${accent}1a`,
							color: accent,
							display: 'flex',
							alignItems: 'center',
							justifyContent: 'center',
						}}
					>
						<EquipmentKindIcon isKit={node.isKit} kind={node.typeKind} sx={{ fontSize: embedded ? 16 : 18 }} />
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
									fontSize: embedded ? '0.8125rem' : '0.875rem',
									// Название получает всю ширину и переносится максимум на две строки.
									display: '-webkit-box',
									WebkitLineClamp: 2,
									WebkitBoxOrient: 'vertical',
									overflow: 'hidden',
									wordBreak: 'break-word',
								}}
							>
								{node.name}
							</Typography>
						</Tooltip>
						{hasToggle && !isExpanded && (
							<Typography component='span' sx={{ fontSize: '0.75rem', color: '#9ca3af', fontWeight: 500, flexShrink: 0 }}>
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
						{subtitle}
					</Typography>

					<Stack direction='row' spacing={0.75} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap', mt: 0.75 }}>
						{node.isKit && (
							<Box
								component='span'
								sx={{
									display: 'inline-flex',
									alignItems: 'center',
									gap: 0.5,
									px: 1,
									py: 0.25,
									borderRadius: '999px',
									fontSize: '0.7rem',
									fontWeight: 600,
									bgcolor: '#f3e8ff',
									color: '#7c3aed',
								}}
							>
								<BoxesIcon sx={{ fontSize: 12 }} />
								Комплект
							</Box>
						)}
						<EquipmentStatusBadge status={node.status} />
						{holder && (
							<Box
								component='span'
								sx={{
									display: 'inline-flex',
									alignItems: 'center',
									px: 1,
									py: 0.25,
									borderRadius: '999px',
									fontSize: '0.7rem',
									fontWeight: 500,
									bgcolor: '#eff6ff',
									color: '#2563eb',
									maxWidth: '100%',
									overflow: 'hidden',
									textOverflow: 'ellipsis',
									whiteSpace: 'nowrap',
								}}
							>
								{holder}
							</Box>
						)}
					</Stack>
				</Box>
			</Stack>

			{/* Раскрытые потомки — внутри той же карточки, компактным списком с линией-связкой. */}
			{children && <Box sx={nestedListSx}>{children}</Box>}
		</Box>
	)
}
