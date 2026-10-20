import type { ElementType } from 'react'
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
	PackageOpenIcon,
	PcCaseIcon,
	PhoneCallIcon,
	PrinterIcon,
} from 'lucide-mui'

// Реестр иконок типов оборудования. Единственный источник иконок и их
// цветов: настройки типов, список, карточка и деталь рисуют отсюда.

export type TypeIconKey =
	| 'laptop'
	| 'monitor'
	| 'printer'
	| 'hard-drive'
	| 'phone'
	| 'memory'
	| 'cpu'
	| 'mouse'
	| 'keyboard'
	| 'briefcase'
	| 'pc-case'
	| 'droplet'
	| 'cable'
	| 'battery'
	| 'package'

export interface TypeIconDef {
	Icon: ElementType
	color: string
	bg: string
}

export const TYPE_ICONS: Record<TypeIconKey, TypeIconDef> = {
	laptop: { Icon: LaptopIcon, color: '#0432a5', bg: '#eaeefc' },
	monitor: { Icon: MonitorIcon, color: '#6b7280', bg: '#f3f4f6' },
	printer: { Icon: PrinterIcon, color: '#6b7280', bg: '#f3f4f6' },
	'hard-drive': { Icon: HardDriveIcon, color: '#9333ea', bg: '#f3e8ff' },
	phone: { Icon: PhoneCallIcon, color: '#6b7280', bg: '#f3f4f6' },
	memory: { Icon: MemoryStickIcon, color: '#ea580c', bg: '#fff7ed' },
	cpu: { Icon: CpuIcon, color: '#ea580c', bg: '#fff7ed' },
	mouse: { Icon: MouseIcon, color: '#6b7280', bg: '#f3f4f6' },
	keyboard: { Icon: KeyboardIcon, color: '#6b7280', bg: '#f3f4f6' },
	briefcase: { Icon: BriefcaseIcon, color: '#a16207', bg: '#fef3c7' },
	'pc-case': { Icon: PcCaseIcon, color: '#374151', bg: '#f3f4f6' },
	droplet: { Icon: DropletIcon, color: '#0891b2', bg: '#ecfeff' },
	cable: { Icon: CableIcon, color: '#6b7280', bg: '#f3f4f6' },
	battery: { Icon: BatteryIcon, color: '#16a34a', bg: '#ecfdf5' },
	package: { Icon: PackageIcon, color: '#6b7280', bg: '#f3f4f6' },
}

export const TYPE_ICON_KEYS = Object.keys(TYPE_ICONS) as TypeIconKey[]

// Правила режима «Авто»: иконка и фон подбираются одним массивом (раньше были
// двумя функциями и разошлись — например, у английского «Laptop» фон оставался
// серым). Порядок правил значим: первое совпадение побеждает.
const ICON_RULES: Array<{ key: TypeIconKey; words: string[] }> = [
	{ key: 'laptop', words: ['ноут', 'laptop'] },
	{ key: 'monitor', words: ['монит', 'диспл', 'monitor'] },
	{ key: 'printer', words: ['принт', 'мфу', 'print'] },
	{ key: 'hard-drive', words: ['накопит', 'диск', 'ssd', 'hdd', 'nvme', 'hard'] },
	{ key: 'phone', words: ['тел', 'смартф', 'phone'] },
	{ key: 'memory', words: ['ram', 'памя', 'memory', 'озу'] },
	{ key: 'cpu', words: ['cpu', 'проц', 'микро'] },
	{ key: 'mouse', words: ['мыш', 'mouse'] },
	{ key: 'keyboard', words: ['клав', 'keyboard'] },
	{ key: 'briefcase', words: ['сумк', 'bag'] },
	{ key: 'pc-case', words: ['системн', 'корпус', 'блок'] },
	{ key: 'droplet', words: ['картридж', 'тонер', 'чернил'] },
	{ key: 'cable', words: ['кабел', 'провод'] },
	{ key: 'battery', words: ['батаре', 'аккумул'] },
]

// guessIconKey подбирает ключ по названию — режим «Авто» для типа без явно
// сохранённой иконки.
export const guessIconKey = (name: string): TypeIconKey => {
	const value = name?.toLowerCase() ?? ''
	const rule = ICON_RULES.find(r => r.words.some(w => value.includes(w)))
	return rule?.key ?? 'package'
}

type ResolveInput = {
	isKit?: boolean
	// icon — сохранённый ключ типа; неизвестные значения деградируют в «Авто».
	icon?: string
	typeName?: string
}

// resolveTypeIcon — единая цепочка подбора: комплект получает отдельную
// иконку, у единицы — явный ключ её типа, без ключа («Авто») — подбор по
// названию, затем общий «Пакет».
export const resolveTypeIcon = ({ isKit, icon, typeName }: ResolveInput): TypeIconDef => {
	if (isKit) return { ...TYPE_ICONS.package, Icon: PackageOpenIcon }
	if (icon && icon in TYPE_ICONS) return TYPE_ICONS[icon as TypeIconKey]
	return TYPE_ICONS[guessIconKey(typeName ?? '')]
}
