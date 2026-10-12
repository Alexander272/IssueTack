import { type FC, useState } from 'react'
import { Button, Menu, MenuItem } from '@mui/material'
import { ChevronDownIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { EquipmentStatus, IEquipment } from '../types/equipment'
import { EQUIPMENT_STATUS_OPTIONS } from '../types/equipment'
import { useChangeEquipmentStatusMutation } from '../equipmentApiSlice'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	unit: IEquipment
}

export const EquipmentStatusMenu: FC<Props> = ({ unit }) => {
	const [anchor, setAnchor] = useState<HTMLElement | null>(null)
	const [pending, setPending] = useState<EquipmentStatus | null>(null)
	const [change, { isLoading }] = useChangeEquipmentStatusMutation()

	const apply = async (status: EquipmentStatus) => {
		try {
			await change({ id: unit.id, body: { status } }).unwrap()
			toast.success('Статус обновлён')
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при смене статуса', { autoClose: false })
		}
	}

	const handleSelect = (status: EquipmentStatus, requiresConfirm: boolean) => {
		setAnchor(null)
		if (requiresConfirm) {
			setPending(status)
			return
		}
		void apply(status)
	}

	return (
		<>
			<Button
				variant='outlined'
				endIcon={<ChevronDownIcon sx={{ fontSize: 16 }} />}
				onClick={e => setAnchor(e.currentTarget)}
				sx={{ borderRadius: '8px', textTransform: 'none' }}
			>
				Изменить статус
			</Button>

			<Menu anchorEl={anchor} open={Boolean(anchor)} onClose={() => setAnchor(null)}>
				{EQUIPMENT_STATUS_OPTIONS.filter(o => o.value !== unit.status).map(o => (
					<MenuItem
						key={o.value}
						onClick={() => handleSelect(o.value, o.value === 'written_off')}
						sx={{ fontSize: 14 }}
					>
						{o.label}
					</MenuItem>
				))}
			</Menu>

			<ConfirmDialog
				open={Boolean(pending)}
				title='Списание оборудования'
				message='Списанное оборудование нельзя изменить. Продолжить?'
				confirmLabel='Списать'
				confirmColor='error'
				onConfirm={() => {
					if (pending) void apply(pending)
					setPending(null)
				}}
				onCancel={() => setPending(null)}
				loading={isLoading}
			/>
		</>
	)
}
