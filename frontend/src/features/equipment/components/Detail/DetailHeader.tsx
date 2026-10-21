import { useState, type FC } from 'react'
import { Box, Button, IconButton, ListItemIcon, ListItemText, Menu, MenuItem, Stack, Typography } from '@mui/material'
import { AlertTriangleIcon, MoreVerticalIcon, PencilIcon, WrenchIcon, XIcon } from 'lucide-mui'
import { useSelector } from 'react-redux'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipment } from '../../types/equipment'
import { HOLDER_DERIVED_STATUSES, getEquipmentStatus } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { EquipmentDefectDialog, EquipmentDialog, EquipmentKitDialog } from '../Dialogs'
import { EquipmentTypeIcon } from '../EquipmentTypeIcon'
import { useWriteOffEquipmentMutation } from '../../equipmentApiSlice'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { getCurrentCapabilities } from '@/features/user/userSlice'

type Props = {
	unit: IEquipment
	onClose: () => void
}

// Шапка модалки единицы: иконка типа и название со статусом, меню действий
// (редактировать, добавить дефект, списать) и кнопка закрытия. Выдача и возврат
// живут в блоке «Держатель» на вкладке «Обзор» — там видно, у кого единица
// сейчас, — а удаление переехало в форму редактирования, рядом с сохранением.
// Диалоги открываются здесь же, рядом с меню, поэтому наружу уходят только
// единица и закрытие модалки — мостиков-колбэков в контейнер нет. Прав на запись
// нужны только здесь, поэтому компонент берёт их сам.
export const DetailHeader: FC<Props> = ({ unit, onClose }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	// Списание необратимо, поэтому доступно только начальнику области — та же
	// формула, что на бэкенде (IsRealmSupervisor), иначе кнопка обещала бы то,
	// чего сервис не разрешит.
	const isRealmAdmin = useSelector(getCurrentCapabilities).isRealmAdmin
	const [writeOff, { isLoading: isWritingOff }] = useWriteOffEquipmentMutation()

	const [editOpen, setEditOpen] = useState(false)
	const [defectOpen, setDefectOpen] = useState(false)
	const [writeOffOpen, setWriteOffOpen] = useState(false)
	const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null)

	// Дефект заводят только на активной единице и не на комплекте: у комплекта нет
	// своего состояния, а у уже дефектной/списанной/в ремонте дефект уже есть —
	// вторая запись «что сломалось» только запутала бы историю.
	const canAddDefect = !unit.isKit && (unit.status === 'in_stock' || unit.status === 'in_use')
	// Списывать можно только дефектную и только начальнику области.
	const canWriteOff = isRealmAdmin && unit.status === 'defective'

	const accent = equipmentAccent(unit)
	// Чипы в шапке — про исключение, а не пересказ карточки: «на складе»/«в
	// эксплуатации» видно в блоке «Держатель» ниже, а ремонт, дефект, резерв и
	// списание больше нигде в карточке не видны — их оставляем.
	const status = HOLDER_DERIVED_STATUSES.includes(unit.status) ? null : getEquipmentStatus(unit.status)

	const writeOffHandler = async () => {
		try {
			await writeOff({ id: unit.id, body: {} }).unwrap()
			toast.success('Оборудование списано')
			setWriteOffOpen(false)
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при списании', { autoClose: false })
		}
	}

	const closeMenu = () => setMenuAnchor(null)

	return (
		<>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					gap: { xs: 1.5, sm: 2 },
					// На телефоне шапка переносится: название и крестик остаются в первой
					// строке, а «Редактировать» уходит отдельной строкой во всю ширину
					// (иначе кнопка сжимала бы название до пары букв). На широком экране
					// порядок и вид прежние: `order`/`flexBasis` сбрасываются.
					flexWrap: { xs: 'wrap', sm: 'nowrap' },
				}}
			>
				<Box
					sx={{
						// На телефоне иконку типа прячем целиком: место в первой строке
						// нужнее названию и статусам, а сам тип виден в характеристиках.
						display: { xs: 'none', sm: 'flex' },
						width: 48,
						height: 48,
						borderRadius: '12px',
						bgcolor: `${accent}1a`,
						color: accent,
						alignItems: 'center',
						justifyContent: 'center',
						flexShrink: 0,
					}}
				>
					<EquipmentTypeIcon isKit={unit.isKit} icon={unit.typeIcon} typeName={unit.typeName} sx={{ fontSize: 24 }} />
				</Box>
				<Box sx={{ flexGrow: 1, minWidth: 0, flexBasis: { xs: 0, sm: 'auto' } }}>
					<Stack direction='row' spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
						<Typography variant='h6' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
							{unit.name}
						</Typography>
						{status && <StatusChip label={status.label} color={status.color} />}
						{unit.isKit && <StatusChip label='Комплект' color={accent} />}
					</Stack>
					{/* Тип, инвентарный и серийный номера на телефоне скрыты: строка
					    отжимала название, а те же номера есть в характеристиках.
					    У комплекта типа нет — его имя в подпись не подмешиваем
					    (тип мог остаться только от старых данных). */}
					<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.25, display: { xs: 'none', sm: 'block' } }}>
						{unit.isKit ? '' : unit.typeName}
						{unit.inventoryNumber ? ` · Инв. №${unit.inventoryNumber}` : ''}
						{unit.serialNumber ? ` · S/N: ${unit.serialNumber}` : ''}
					</Typography>
				</Box>
				{canWrite && (
					<Stack
						direction='row'
						spacing={1}
						sx={{
							flexWrap: 'wrap',
							gap: 1,
							alignItems: 'center',
							order: { xs: 3, sm: 0 },
							flexBasis: { xs: '100%', sm: 'auto' },
						}}
					>
						{/* Действия собраны в меню под тремя точками: «Редактировать»
						    рядом с «Добавить дефект» и «Списать» — это операции над
						    одной единицей, и держать их в одном месте логичнее, чем
						    растить ряд кнопок в шапке. На телефоне кнопка занимает
						    строку целиком (как раньше «Редактировать»). */}
						<Button
							variant='outlined'
							size='small'
							onClick={e => setMenuAnchor(e.currentTarget)}
							sx={{ borderRadius: '8px', textTransform: 'none', gap: 0.5 }}
						>
							<MoreVerticalIcon sx={{ fontSize: 16 }} />
							Действия
						</Button>
						<Menu
							anchorEl={menuAnchor}
							open={Boolean(menuAnchor)}
							onClose={closeMenu}
							slotProps={{ paper: { sx: { minWidth: 200 } } }}
						>
							<MenuItem
								onClick={() => {
									closeMenu()
									setEditOpen(true)
								}}
							>
								<ListItemIcon>
									<PencilIcon sx={{ fontSize: 18 }} />
								</ListItemIcon>
								<ListItemText>Редактировать</ListItemText>
							</MenuItem>
							{canAddDefect && (
								<MenuItem
									onClick={() => {
										closeMenu()
										setDefectOpen(true)
									}}
								>
									<ListItemIcon>
										<WrenchIcon sx={{ fontSize: 18, color: 'warning.main' }} />
									</ListItemIcon>
									<ListItemText>Добавить дефект</ListItemText>
								</MenuItem>
							)}
							{canWriteOff && (
								<MenuItem
									onClick={() => {
										closeMenu()
										setWriteOffOpen(true)
									}}
								>
									<ListItemIcon>
										<AlertTriangleIcon sx={{ fontSize: 18, color: 'error.main' }} />
									</ListItemIcon>
									<ListItemText sx={{ color: 'error.main' }}>Списать</ListItemText>
								</MenuItem>
							)}
						</Menu>
					</Stack>
				)}
				<IconButton
					size='large'
					onClick={onClose}
					sx={{ color: '#9ca3af', order: { xs: 2, sm: 0 }, flexShrink: 0 }}
				>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</Box>

			{/* Условный рендер — форма монтируется заново при каждом открытии,
			    чтобы не показывать значения предыдущего элемента. Комплект
			    правится той же формой, что создавался (EquipmentKitDialog),
			    единица — EquipmentDialog. */}
			{editOpen &&
				(unit.isKit ? (
					<EquipmentKitDialog open onClose={() => setEditOpen(false)} onDeleted={onClose} unit={unit} />
				) : (
					<EquipmentDialog open onClose={() => setEditOpen(false)} onDeleted={onClose} unit={unit} />
				))}

			{/* Условный рендер дефекта: форма монтируется заново при каждом
			    открытии, чтобы не показывать описание предыдущей единицы. */}
			{defectOpen && <EquipmentDefectDialog open onClose={() => setDefectOpen(false)} unit={unit} />}

			<ConfirmDialog
				open={writeOffOpen}
				title='Списать оборудование'
				message={`«${unit.name}» будет списано. Действие необратимо, единица уйдёт из общего списка.`}
				confirmLabel='Списать'
				confirmColor='error'
				loading={isWritingOff}
				loadingLabel='Списание...'
				onConfirm={writeOffHandler}
				onCancel={() => setWriteOffOpen(false)}
			/>
		</>
	)
}

const StatusChip: FC<{ label: string; color: string }> = ({ label, color }) => (
	<Box
		component='span'
		sx={{
			px: 1,
			py: 0.25,
			borderRadius: '6px',
			fontSize: 12,
			fontWeight: 600,
			bgcolor: `${color}1a`,
			color,
			whiteSpace: 'nowrap',
		}}
	>
		{label}
	</Box>
)
