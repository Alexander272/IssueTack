import { useState, type FC } from 'react'
import { Box, Button, Stack, Typography } from '@mui/material'
import { PackageCheckIcon, PackageMinusIcon, WarehouseIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import type { IEquipment } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { useUnassignEquipmentMutation } from '../../equipmentApiSlice'
import { EquipmentAssignDialog } from '../Dialogs'
import { DetailCard } from './DetailCard'
import { sectionLabelSx, softButtonSx } from './consts'

type Props = {
	unit: IEquipment
}

// Блок «Держатель»: сотрудник (или «На складе») и площадка, а справа — выдача и
// возврат. Права и диалог выдачи нужны только здесь, поэтому карточка берёт их
// сама. Установленную единицу отдельно не выдают: она уже внутри устройства или
// комплекта.
export const HolderCard: FC<Props> = ({ unit }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const [assignOpen, setAssignOpen] = useState(false)
	const [unassign, { isLoading: isUnassigning }] = useUnassignEquipmentMutation()

	const accent = equipmentAccent(unit)
	const holder = unit.holderFullName || unit.holderName || ''
	const hasHolder = Boolean(unit.holderUserId || unit.holderName)
	const installed = Boolean(unit.parentId || unit.kitId)
	const initials = holder
		.split(' ')
		.map(s => s[0])
		.join('')
		.slice(0, 2)

	const handleUnassign = async () => {
		try {
			// Комментарий не передаём: возврат описывает себя сам (кто, когда, где),
			// а любая заготовка вроде «Возврат со страницы оборудования» оседала бы в
			// примечании строки истории и в акте как мусор.
			await unassign({ id: unit.id, body: {} }).unwrap()
			toast.success('Оборудование возвращено')
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	return (
		<>
			<DetailCard>
				<Box
					sx={{
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'space-between',
						gap: 2,
						py: 0.5,
					}}
				>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.75, minWidth: 0 }}>
						<Box
							sx={{
								width: 40,
								height: 40,
								borderRadius: '12px',
								flexShrink: 0,
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'center',
								fontSize: 14,
								fontWeight: 600,
								...(hasHolder
									? { bgcolor: `${accent}1a`, color: accent }
									: { bgcolor: '#f1f5f9', color: '#64748b' }),
							}}
						>
							{hasHolder ? initials : <WarehouseIcon sx={{ fontSize: 20 }} />}
						</Box>
						<Box sx={{ minWidth: 0 }}>
							<Typography sx={sectionLabelSx}>Держатель</Typography>
							<Typography sx={{ fontSize: 14, fontWeight: 600, color: '#0f172a' }} noWrap>
								{hasHolder ? holder : 'На складе'}
							</Typography>
							<Typography sx={{ fontSize: 12, color: '#64748b' }} noWrap>
								{unit.siteName || unit.location || '—'}
							</Typography>
						</Box>
					</Box>
					<Stack direction='row' spacing={1} sx={{ alignItems: 'center', flexShrink: 0 }}>
						{!hasHolder && (
							<Box
								component='span'
								sx={{
									display: 'inline-flex',
									alignItems: 'center',
									gap: 0.75,
									px: 1.25,
									py: 0.5,
									borderRadius: '8px',
									border: '1px solid #a7f3d0',
									bgcolor: '#ecfdf5',
									color: '#047857',
									fontSize: 12,
									fontWeight: 500,
									whiteSpace: 'nowrap',
								}}
							>
								<Box
									component='span'
									sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: '#10b981' }}
								/>
								Доступно
							</Box>
						)}
						{canWrite && !installed && (
							<Button
								size='small'
								disabled={isUnassigning}
								startIcon={
									hasHolder ? (
										<PackageMinusIcon sx={{ fontSize: 14 }} />
									) : (
										<PackageCheckIcon sx={{ fontSize: 14 }} />
									)
								}
								sx={softButtonSx}
								onClick={() => (hasHolder ? handleUnassign() : setAssignOpen(true))}
							>
								{hasHolder ? 'Возврат' : 'Выдать'}
							</Button>
						)}
					</Stack>
				</Box>
			</DetailCard>

			{/* Форма выдачи монтируется заново при каждом открытии, чтобы не
			    показывать значения предыдущей единицы. */}
			{assignOpen && <EquipmentAssignDialog open onClose={() => setAssignOpen(false)} unit={unit} />}
		</>
	)
}
