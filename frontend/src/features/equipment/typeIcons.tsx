import type { ElementType } from 'react'
import {
	BatteryIcon,
	BriefcaseIcon,
	CableIcon,
	CameraIcon,
	CctvIcon,
	CpuIcon,
	DropletIcon,
	FanIcon,
	HardDriveIcon,
	HeadphonesIcon,
	KeyboardIcon,
	LaptopIcon,
	MemoryStickIcon,
	MicIcon,
	MonitorIcon,
	MouseIcon,
	PackageIcon,
	PackageOpenIcon,
	PcCaseIcon,
	PhoneCallIcon,
	PlugZapIcon,
	PrinterIcon,
	ProjectorIcon,
	RouterIcon,
	ScanLineIcon,
	ServerIcon,
	SmartphoneIcon,
	SpeakerIcon,
	TabletIcon,
	TvIcon,
	UsbIcon,
	WebcamIcon,
} from 'lucide-mui'

// Реестр иконок типов оборудования. Единственный источник иконок и их
// цветов: настройки типов, список, карточка и деталь рисуют отсюда.

export type TypeIconKey =
	| 'laptop'
	| 'monitor'
	| 'printer'
	| 'scanner'
	| 'projector'
	| 'tv'
	| 'hard-drive'
	| 'server'
	| 'router'
	| 'phone'
	| 'tablet'
	| 'smartphone'
	| 'memory'
	| 'cpu'
	| 'usb'
	| 'mouse'
	| 'keyboard'
	| 'webcam'
	| 'camera'
	| 'cctv'
	| 'speaker'
	| 'headphones'
	| 'mic'
	| 'briefcase'
	| 'pc-case'
	| 'droplet'
	| 'cable'
	| 'battery'
	| 'power'
	| 'fan'
	| 'package'

export interface TypeIconDef {
	Icon: ElementType
	// label — что за сущность обозначает иконка. Показывается подсказкой в
	// выборе иконки: ключи вроде `router`/`cctv` сами по себе не читаются.
	label: string
	color: string
	bg: string
}

export const TYPE_ICONS: Record<TypeIconKey, TypeIconDef> = {
	laptop: { Icon: LaptopIcon, label: 'Ноутбук', color: '#0432a5', bg: '#eaeefc' },
	monitor: { Icon: MonitorIcon, label: 'Монитор', color: '#6b7280', bg: '#f3f4f6' },
	printer: { Icon: PrinterIcon, label: 'Принтер / МФУ', color: '#6b7280', bg: '#f3f4f6' },
	scanner: { Icon: ScanLineIcon, label: 'Сканер', color: '#6b7280', bg: '#f3f4f6' },
	projector: { Icon: ProjectorIcon, label: 'Проектор', color: '#6b7280', bg: '#f3f4f6' },
	tv: { Icon: TvIcon, label: 'Телевизор / экран', color: '#6b7280', bg: '#f3f4f6' },
	'hard-drive': { Icon: HardDriveIcon, label: 'Накопитель (SSD/HDD)', color: '#9333ea', bg: '#f3e8ff' },
	server: { Icon: ServerIcon, label: 'Сервер', color: '#374151', bg: '#f3f4f6' },
	router: { Icon: RouterIcon, label: 'Роутер / коммутатор', color: '#0891b2', bg: '#ecfeff' },
	phone: { Icon: PhoneCallIcon, label: 'Телефон (IP)', color: '#6b7280', bg: '#f3f4f6' },
	tablet: { Icon: TabletIcon, label: 'Планшет', color: '#6b7280', bg: '#f3f4f6' },
	smartphone: { Icon: SmartphoneIcon, label: 'Смартфон', color: '#0891b2', bg: '#ecfeff' },
	memory: { Icon: MemoryStickIcon, label: 'Оперативная память', color: '#ea580c', bg: '#fff7ed' },
	cpu: { Icon: CpuIcon, label: 'Процессор', color: '#ea580c', bg: '#fff7ed' },
	usb: { Icon: UsbIcon, label: 'USB / флешка', color: '#6b7280', bg: '#f3f4f6' },
	mouse: { Icon: MouseIcon, label: 'Мышь', color: '#6b7280', bg: '#f3f4f6' },
	keyboard: { Icon: KeyboardIcon, label: 'Клавиатура', color: '#6b7280', bg: '#f3f4f6' },
	webcam: { Icon: WebcamIcon, label: 'Веб-камера', color: '#6b7280', bg: '#f3f4f6' },
	camera: { Icon: CameraIcon, label: 'Фотоаппарат', color: '#6b7280', bg: '#f3f4f6' },
	cctv: { Icon: CctvIcon, label: 'Видеонаблюдение', color: '#6b7280', bg: '#f3f4f6' },
	speaker: { Icon: SpeakerIcon, label: 'Колонки', color: '#6b7280', bg: '#f3f4f6' },
	headphones: { Icon: HeadphonesIcon, label: 'Наушники / гарнитура', color: '#6b7280', bg: '#f3f4f6' },
	mic: { Icon: MicIcon, label: 'Микрофон', color: '#6b7280', bg: '#f3f4f6' },
	briefcase: { Icon: BriefcaseIcon, label: 'Сумка / кейс', color: '#a16207', bg: '#fef3c7' },
	'pc-case': { Icon: PcCaseIcon, label: 'Системный блок / корпус', color: '#374151', bg: '#f3f4f6' },
	droplet: { Icon: DropletIcon, label: 'Картридж / чернила', color: '#0891b2', bg: '#ecfeff' },
	cable: { Icon: CableIcon, label: 'Кабель / переходник', color: '#6b7280', bg: '#f3f4f6' },
	battery: { Icon: BatteryIcon, label: 'Батарея / аккумулятор', color: '#16a34a', bg: '#ecfdf5' },
	power: { Icon: PlugZapIcon, label: 'ИБП / питание', color: '#dc2626', bg: '#fef2f2' },
	fan: { Icon: FanIcon, label: 'Кулер / охлаждение', color: '#0891b2', bg: '#ecfeff' },
	package: { Icon: PackageIcon, label: 'Общее (по умолчанию)', color: '#6b7280', bg: '#f3f4f6' },
}

export const TYPE_ICON_KEYS = Object.keys(TYPE_ICONS) as TypeIconKey[]

// Правила режима «Авто»: иконка и фон подбираются одним массивом (раньше были
// двумя функциями и разошлись — например, у английского «Laptop» фон оставался
// серым). Порядок правил значим: первое совпадение побеждает.
const ICON_RULES: Array<{ key: TypeIconKey; words: string[] }> = [
	{ key: 'laptop', words: ['ноут', 'laptop'] },
	{ key: 'monitor', words: ['монит', 'диспл', 'monitor'] },
	{ key: 'webcam', words: ['веб-камер', 'вебкам', 'webcam'] },
	{ key: 'cctv', words: ['видеонаблюд', 'камер видео', 'cctv'] },
	{ key: 'camera', words: ['фотоаппар', 'камер'] },
	{ key: 'scanner', words: ['сканер', 'scan'] },
	{ key: 'printer', words: ['принт', 'мфу', 'print'] },
	{ key: 'projector', words: ['проектор', 'projector'] },
	{ key: 'tv', words: ['телевиз', 'tv '] },
	{ key: 'server', words: ['сервер', 'server'] },
	{ key: 'router', words: ['роутер', 'маршрутиз', 'коммутатор', 'маршрут', 'router', 'switch'] },
	{ key: 'phone', words: ['телефон', 'ip-тел', 'ip-фон', 'phone'] },
	{ key: 'smartphone', words: ['смартфон', 'smartphone'] },
	{ key: 'tablet', words: ['планшет', 'tablet'] },
	{ key: 'hard-drive', words: ['накопит', 'диск', 'ssd', 'hdd', 'nvme', 'hard'] },
	{ key: 'memory', words: ['ram', 'памя', 'memory', 'озу'] },
	{ key: 'cpu', words: ['cpu', 'проц', 'микро', 'материнск', 'видеокарт'] },
	{ key: 'usb', words: ['usb', 'флеш', 'флаш', 'контроллер'] },
	{ key: 'headphones', words: ['наушник', 'гарнитур', 'headphone'] },
	{ key: 'speaker', words: ['колонк', 'динамик', 'speaker'] },
	{ key: 'mic', words: ['микрофон', 'microphone'] },
	{ key: 'mouse', words: ['мыш', 'mouse'] },
	{ key: 'keyboard', words: ['клав', 'keyboard'] },
	{ key: 'briefcase', words: ['сумк', 'bag'] },
	{ key: 'power', words: ['ибп', 'ups', 'бесперебой', 'питани', 'зарядн'] },
	{ key: 'pc-case', words: ['системн', 'корпус', 'блок'] },
	{ key: 'fan', words: ['кулер', 'вентилятор', 'охлажд'] },
	{ key: 'battery', words: ['батаре', 'аккумул'] },
	{ key: 'droplet', words: ['картридж', 'тонер', 'чернил'] },
	{ key: 'cable', words: ['кабел', 'провод', 'переходник', 'адаптер', 'патч-корд'] },
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
	if (isKit) return { ...TYPE_ICONS.package, Icon: PackageOpenIcon, label: 'Комплект' }
	if (icon && icon in TYPE_ICONS) return TYPE_ICONS[icon as TypeIconKey]
	return TYPE_ICONS[guessIconKey(typeName ?? '')]
}
