import { useState, type FC } from 'react'
import { Box, Button, Typography, useMediaQuery, useTheme } from '@mui/material'
import { LayersIcon, PlusIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipmentKitTemplate } from '../../types/equipment'
import { useDeleteEquipmentTemplateMutation, useGetAllEquipmentTemplatesQuery } from '../../equipmentApiSlice'
import { EquipmentTemplateDialog } from '../Dialogs'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'
import { EquipmentTemplateRow } from './EquipmentTemplateRow'

// Секция настроек «Шаблоны комплектов»: именованные наборы типов оборудования,
// из которых собирается состав комплекта. Блок «Собрать из шаблона» в модалке
// создания комплекта показывается только когда здесь есть хотя бы один шаблон.
export const EquipmentTemplatesSection: FC = () => {
	const theme = useTheme()
	// Строка шаблона — 12-колоночная сетка: на телефоне она нечитаема, поэтому там
	// строка рисуется карточкой. Режим считаем один раз и раздаём пропсом.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })

	const { data } = useGetAllEquipmentTemplatesQuery()
	const [remove] = useDeleteEquipmentTemplateMutation()

	const [dialogOpen, setDialogOpen] = useState(false)
	const [editTemplate, setEditTemplate] = useState<IEquipmentKitTemplate | null>(null)
	const [deleteTarget, setDeleteTarget] = useState<IEquipmentKitTemplate | null>(null)

	const templates = data?.data ?? []

	const openCreate = () => {
		setEditTemplate(null)
		setDialogOpen(true)
	}

	const openEdit = (template: IEquipmentKitTemplate) => {
		setEditTemplate(template)
		setDialogOpen(true)
	}

	const handleDelete = async () => {
		if (!deleteTarget?.id) return
		try {
			await remove(deleteTarget.id).unwrap()
			toast.success('Шаблон удалён')
			setDeleteTarget(null)
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при удалении', { autoClose: false })
		}
	}

	return (
		<Box>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'flex-end',
					mb: { xs: 2, sm: -3 },
				}}
			>
				{canWrite && (
					<Button
						variant='outlined'
						onClick={openCreate}
						startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
						sx={{
							textTransform: 'none',
							fontWeight: 500,
							borderRadius: '8px',
							gap: 1,
							bgcolor: '#fff',
							width: { xs: '100%', sm: 'auto' },
						}}
					>
						Добавить шаблон
					</Button>
				)}
			</Box>

			{templates.length === 0 ? (
				<Box
					sx={{
						display: 'flex',
						flexDirection: 'column',
						alignItems: 'center',
						gap: 1,
						py: 2,
						px: 2,
						mt: 5,
						bgcolor: '#fff',
						border: '1px dashed #d1d5db',
						borderRadius: '12px',
						textAlign: 'center',
					}}
				>
					<LayersIcon sx={{ fontSize: 32, color: '#9ca3af' }} />
					<Typography variant='body2' sx={{ fontWeight: 600, color: '#374151' }}>
						Шаблонов комплектов пока нет
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						Шаблон — это набор типов оборудования, из которого собирается состав комплекта
					</Typography>
				</Box>
			) : (
				<Box>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
						<Typography
							variant='subtitle2'
							sx={{ color: '#374151', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.3 }}
						>
							Шаблоны
						</Typography>
						<Typography variant='caption' sx={{ color: '#6b7280' }}>
							· {templates.length} {pluralize(templates.length, ['шаблон', 'шаблона', 'шаблонов'])}
						</Typography>
					</Box>
					<Box
						sx={{
							bgcolor: '#fff',
							border: '1px solid #e5e7eb',
							borderRadius: '12px',
							boxShadow: '0 1px 2px 0 rgba(0,0,0,0.05)',
							// Чтобы подсветка строки на ховере не вылезала за скруглённые углы.
							overflow: 'hidden',
						}}
					>
						{templates.map(template => (
							<EquipmentTemplateRow
								key={template.id}
								template={template}
								onEdit={openEdit}
								onDelete={setDeleteTarget}
								mobile={isMobile}
							/>
						))}
					</Box>
				</Box>
			)}

			<EquipmentTemplateDialog
				key={editTemplate?.id ?? 'new'}
				open={dialogOpen}
				onClose={() => {
					setEditTemplate(null)
					setDialogOpen(false)
				}}
				template={editTemplate}
			/>

			<ConfirmDialog
				open={Boolean(deleteTarget)}
				title='Удаление шаблона комплекта'
				message={`Удалить шаблон «${deleteTarget?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteTarget(null)}
			/>
		</Box>
	)
}
