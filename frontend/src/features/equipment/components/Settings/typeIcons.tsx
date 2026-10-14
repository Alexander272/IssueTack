import {
	BatteryIcon,
	BriefcaseIcon,
	CableIcon,
	CpuIcon,
	DropletIcon,
	HardDriveIcon,
	KeyboardIcon,
	LaptopIcon,
	MemoryStickIcon,
	MonitorIcon,
	MouseIcon,
	PackageIcon,
	PcCaseIcon,
	PhoneCallIcon,
	PrinterIcon,
} from 'lucide-mui'

import type { EquipmentTypeKind, IEquipmentType } from '../../types/equipment'

// Группа типов — это категория типа; порядок и подписи задают разбивку секции.
export type GroupKey = EquipmentTypeKind

export const GROUP_ORDER: GroupKey[] = ['device', 'component', 'periphery', 'consumable']

export const GROUP_LABELS: Record<GroupKey, string> = {
	device: 'Устройства',
	component: 'Комплектующие',
	periphery: 'Периферия',
	consumable: 'Расходники',
}

// Цвета бейджа категории в строке типа.
export const BADGE_STYLES: Record<GroupKey, { bgcolor: string; color: string }> = {
	device: { bgcolor: '#f3f4f6', color: '#374151' },
	component: { bgcolor: '#fff7ed', color: '#9a3412' },
	periphery: { bgcolor: '#eff6ff', color: '#1e40af' },
	consumable: { bgcolor: '#ecfdf5', color: '#065f46' },
}

// Иконка типа подбирается по названию: у типов нет отдельного поля иконки.
export const getIcon = (type: IEquipmentType) => {
	const name = type.name.toLowerCase()
	if (name.includes('ноут') || name.includes('laptop')) return <LaptopIcon sx={{ color: '#0432a5' }} />
	if (name.includes('монит') || name.includes('диспл') || name.includes('monitor'))
		return <MonitorIcon sx={{ color: '#6b7280' }} />
	if (name.includes('принт') || name.includes('мфу') || name.includes('print'))
		return <PrinterIcon sx={{ color: '#6b7280' }} />
	if (
		name.includes('накопит') ||
		name.includes('диск') ||
		name.includes('ssd') ||
		name.includes('hdd') ||
		name.includes('nvme') ||
		name.includes('hard')
	)
		return <HardDriveIcon sx={{ color: '#9333ea' }} />
	if (name.includes('тел') || name.includes('смартф') || name.includes('phone'))
		return <PhoneCallIcon sx={{ color: '#6b7280' }} />
	if (name.includes('ram') || name.includes('памя') || name.includes('memory') || name.includes('озу'))
		return <MemoryStickIcon sx={{ color: '#ea580c' }} />

	if (name.includes('cpu') || name.includes('проц') || name.includes('микро'))
		return <CpuIcon sx={{ color: '#ea580c' }} />
	if (name.includes('мыш') || name.includes('mouse')) return <MouseIcon sx={{ color: '#6b7280' }} />
	if (name.includes('клав') || name.includes('keyboard')) return <KeyboardIcon sx={{ color: '#6b7280' }} />
	if (name.includes('сумк') || name.includes('bag')) return <BriefcaseIcon sx={{ color: '#a16207' }} />
	if (name.includes('системн') || name.includes('корпус') || name.includes('блок'))
		return <PcCaseIcon sx={{ color: '#374151' }} />
	if (name.includes('картридж') || name.includes('тонер') || name.includes('чернил'))
		return <DropletIcon sx={{ color: '#0891b2' }} />
	if (name.includes('кабел') || name.includes('провод')) return <CableIcon sx={{ color: '#6b7280' }} />
	if (name.includes('батаре') || name.includes('аккумул')) return <BatteryIcon sx={{ color: '#16a34a' }} />
	return <PackageIcon sx={{ color: '#6b7280' }} />
}

// Фон плитки иконки — тот же подбор по названию, чуть приглушённее.
export const getIconBg = (type: IEquipmentType): string => {
	const name = type.name.toLowerCase()
	if (name.includes('ноут')) return '#eaeefc'
	if (name.includes('ram') || name.includes('памя') || name.includes('memory') || name.includes('озу'))
		return '#fff7ed'
	if (name.includes('накопит') || name.includes('диск') || name.includes('ssd') || name.includes('hdd'))
		return '#f3e8ff'
	if (name.includes('cpu') || name.includes('проц') || name.includes('микро')) return '#fff7ed'
	if (name.includes('сумк')) return '#fef3c7'
	if (name.includes('картридж') || name.includes('тонер') || name.includes('чернил')) return '#ecfeff'
	if (name.includes('батаре') || name.includes('аккумул')) return '#ecfdf5'
	return '#f3f4f6'
}
