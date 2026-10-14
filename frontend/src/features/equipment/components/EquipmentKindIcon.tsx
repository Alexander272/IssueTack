import type { ElementType, FC } from 'react'
import type { SxProps, Theme } from '@mui/material'
import {
	DropletIcon,
	LaptopIcon,
	MemoryStickIcon,
	MouseIcon,
	PackageIcon,
	PackageOpenIcon,
} from 'lucide-mui'

import type { EquipmentTypeKind } from '../types/equipment'

const KIND_ICONS: Record<EquipmentTypeKind, ElementType> = {
	device: LaptopIcon,
	component: MemoryStickIcon,
	periphery: MouseIcon,
	consumable: DropletIcon,
}

type IconProps = {
	isKit?: boolean
	kind?: EquipmentTypeKind
	sx?: SxProps<Theme>
}

// Иконка единицы по категории типа; комплект получает отдельную иконку.
export const EquipmentKindIcon: FC<IconProps> = ({ isKit, kind, sx }) => {
	if (isKit) return <PackageOpenIcon sx={sx} />
	const Icon = kind ? KIND_ICONS[kind] : PackageIcon
	return <Icon sx={sx} />
}
