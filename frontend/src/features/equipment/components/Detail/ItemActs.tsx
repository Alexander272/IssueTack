import { type FC, useMemo, useRef, useState } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableRow,
	TextField,
	Typography,
} from '@mui/material'
import dayjs from 'dayjs'
import { DownloadIcon, EyeIcon, FileSignatureIcon, FileTextIcon, PlusIcon, UploadIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import { Loader } from '@/components/Fallback/Loader'
import type { IFetchError } from '@/app/types/error'
import { saveAs } from '@/utils/saveAs'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import type { EquipmentActStatus, IEquipment, IEquipmentAct } from '../../types/equipment'
import {
	useCreateEquipmentActMutation,
	useGetAllEquipmentActsQuery,
	useGetEquipmentActSignedFileQuery,
	useLazyGetEquipmentActDocumentQuery,
	useLazyGetEquipmentActQuery,
	useLazyGetEquipmentActSignedFileQuery,
	useUploadEquipmentActSignedMutation,
} from '../../equipmentApiSlice'
import { DetailCard } from './DetailCard'
import { EmptyState } from './EmptyState'
import { softButtonSx } from './consts'

// ACT_KIND_LABEL — подпись вида акта в строке списка. Вид акта пока один (выдача),
// но подпись выделена константой: с видами (возврат, списание) она станет полем.
const ACT_KIND_LABEL = 'Выдача'

type ActStatusMeta = { label: string; color: string; bgcolor: string; border: string }

// Статус акта: «сформирован» ждёт подписи (янтарный), «подписан» — файл загружен
// (зелёный). Цвета те же, что у статусов оборудования, — вкладки не выглядят
// разнородными.
const ACT_STATUS_META: Record<EquipmentActStatus, ActStatusMeta> = {
	formed: { label: 'Сформирован', color: '#b45309', bgcolor: '#fef3c7', border: '#fde68a' },
	signed: { label: 'Подписан', color: '#15803d', bgcolor: '#dcfce7', border: '#bbf7d0' },
}

const formatDate = (d: string | null) => (d ? dayjs(d).format('DD.MM.YYYY') : '—')

const customerLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full ? `${full} (${u.username})` : u.username
}

// ActStatusChip — чип состояния акта (как чип «Подписан» в макете).
const ActStatusChip: FC<{ status: EquipmentActStatus }> = ({ status }) => {
	const meta = ACT_STATUS_META[status] ?? ACT_STATUS_META.formed
	return (
		<Box
			component='span'
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				gap: 0.75,
				px: 1.25,
				py: 0.5,
				borderRadius: '8px',
				border: `1px solid ${meta.border}`,
				bgcolor: meta.bgcolor,
				color: meta.color,
				fontSize: 12,
				fontWeight: 500,
				whiteSpace: 'nowrap',
				// Чип не сжимается и не переносится: его текст — nowrap, поэтому
				// при сжатии содержимое вылезало за границы чипа и накладывалось
				// на номер акта. Отдаём сжатие блоку номера (у него minWidth: 0).
				flexShrink: 0,
			}}
		>
			<Box component='span' sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: meta.color }} />
			{meta.label}
		</Box>
	)
}

type RowProps = {
	act: IEquipmentAct
	canWrite: boolean
	onOpen: (act: IEquipmentAct) => void
	onDownload: (act: IEquipmentAct) => void
	onUpload: (act: IEquipmentAct) => void
}

// ActRow — строка списка: иконка, номер, «дата · вид» и действия по статусу.
// Действия гасят всплытие: иначе клик по кнопке заодно открывал бы карточку.
const ActRow: FC<RowProps> = ({ act, canWrite, onOpen, onDownload, onUpload }) => (
	<Box
		onClick={() => onOpen(act)}
		sx={{
			display: 'flex',
			gap: 2,
			p: 1.5,
			border: '1px solid #e9eef5',
			borderRadius: '12px',
			bgcolor: '#f8fafc',
			cursor: 'pointer',
			transition: 'background-color 120ms ease',
			'&:hover': { bgcolor: '#eff6ff' },
			// Перенос разрешён на любой ширине: рядом с иконкой и номером могут
			// не ужиться чип статуса и кнопки, и тогда они уезжают на новую
			// строку. Без переноса чип прижимался к схлопнутому блоку номера,
			// наезжал на «АКТ-…» и обрезал текст в «…».
			flexWrap: 'wrap',
			// Выравнивание по верху, а не по центру: при переносе центр считался
			// бы по высоте всего блока, и иконка с номером уезжали бы вниз, к
			// кнопкам.
			alignItems: { xs: 'flex-start', md: 'center' },
		}}
	>
		<Box
			sx={{
				width: 36,
				height: 36,
				borderRadius: '10px',
				flexShrink: 0,
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
				bgcolor: '#eff6ff',
				color: '#2563eb',
			}}
		>
			<FileTextIcon sx={{ fontSize: 20 }} />
		</Box>

		<Box sx={{ minWidth: 0, flex: '1 1 200px' }}>
			<Typography sx={{ fontSize: 14, fontWeight: 600, color: '#0f172a' }} noWrap>
				{act.number}
			</Typography>
			<Typography sx={{ fontSize: 12, color: '#64748b' }} noWrap>
				{formatDate(act.issuedAt)} · {ACT_KIND_LABEL}
				{act.content?.length ? ` · ${act.content.length} поз.` : ''}
			</Typography>
		</Box>

		<ActStatusChip status={act.status} />

		<Stack
			// direction задаётся пропом, а не через sx: Stack считает spacing по
			// нему (margin-left в строке, margin-top в колонке) — если положить
			// flexDirection в sx, направление для spacing останется undefined.
			direction={{ xs: 'column', md: 'row' }}
			spacing={0.75}
			onClick={e => e.stopPropagation()}
			sx={{
				// До md кнопки стоят друг под другом во всю ширину строки (рядом не
				// уживаются: «Загрузить подписанный» выдавливает «Скачать» и номер
				// акта), на md и шире — компактная строка справа.
				alignItems: { xs: 'stretch', md: 'center' },
				flexShrink: 0,
				order: { xs: 3, md: 0 },
				flexBasis: { xs: '100%', md: 'auto' },
			}}
		>
			{act.status === 'signed' ? (
				<>
					<Button
						size='small'
						startIcon={<EyeIcon sx={{ fontSize: 14 }} />}
						sx={softButtonSx}
						onClick={() => onOpen(act)}
					>
						Просмотреть
					</Button>
					<Button
						size='small'
						startIcon={<DownloadIcon sx={{ fontSize: 14 }} />}
						sx={softButtonSx}
						onClick={() => onDownload(act)}
					>
						Скачать
					</Button>
				</>
			) : (
				<>
					<Button
						size='small'
						startIcon={<DownloadIcon sx={{ fontSize: 14 }} />}
						sx={softButtonSx}
						onClick={() => onDownload(act)}
					>
						Скачать
					</Button>
					{canWrite && (
						<Button
							size='small'
							startIcon={<UploadIcon sx={{ fontSize: 14 }} />}
							sx={softButtonSx}
							onClick={() => onUpload(act)}
						>
							Загрузить подписанный
						</Button>
					)}
				</>
			)}
		</Stack>
	</Box>
)

type Props = {
	unit: IEquipment
}

// ItemActs — вкладка «Акты»: список актов единицы (она может быть комплектом
// выдачи или позицией состава) и подписание. Акт проходит три состояния: docx
// формируется и выгружается, затем загружается скан/фото подписанного акта, у
// подписанного появляется просмотр с возможностью скачать.
export const ItemActs: FC<Props> = ({ unit }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })

	// Фильтр серверный (equipmentId), а не выборка всего реалма с отсевом на
	// клиенте: список актов растёт вместе с парком.
	const filter = useMemo(() => ({ equipmentId: unit.id, limit: 100 }), [unit.id])
	const { data, isLoading, isError } = useGetAllEquipmentActsQuery(filter)
	const acts = data?.data ?? []

	const [detail, setDetail] = useState<IEquipmentAct | null>(null)
	const [createOpen, setCreateOpen] = useState(false)

	const [getDocument] = useLazyGetEquipmentActDocumentQuery()
	const [getSignedFile] = useLazyGetEquipmentActSignedFileQuery()
	const [uploadSigned] = useUploadEquipmentActSignedMutation()

	const fileInputRef = useRef<HTMLInputElement>(null)
	const [uploadTarget, setUploadTarget] = useState<IEquipmentAct | null>(null)

	// Выгрузка сформированного docx: документ собирается на бэкенде по снимку
	// состава, поэтому имя файла берём из номера акта.
	const downloadDocument = async (act: IEquipmentAct) => {
		try {
			const blob = await getDocument(act.id).unwrap()
			saveAs(blob, `${act.number}.docx`)
		} catch {
			toast.error('Не удалось получить акт', { autoClose: false })
		}
	}

	// Скачивание подписанного файла: имя — исходное имя загруженного скана.
	const downloadSigned = async (act: IEquipmentAct) => {
		try {
			const { url } = await getSignedFile(act.id).unwrap()
			const res = await fetch(url)
			if (!res.ok) throw new Error('failed to fetch signed act')
			saveAs(await res.blob(), act.signedFileName || `${act.number}.jpg`)
		} catch (err) {
			const fetchError = err as IFetchError
			toast.error(fetchError.data?.message ?? 'Не удалось получить подписанный акт', { autoClose: false })
		}
	}

	const pickSignedFile = (act: IEquipmentAct) => {
		setUploadTarget(act)
		fileInputRef.current?.click()
	}

	const handleSignedFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0]
		// Значение сбрасываем сразу: иначе повторный выбор того же файла не вызовет
		// change (например, после ошибки загрузки).
		e.target.value = ''
		if (!file || !uploadTarget) return

		try {
			await uploadSigned({ id: uploadTarget.id, file }).unwrap()
			toast.success('Подписанный акт загружен')
		} catch {
			// Ошибку показывает onQueryStarted мутации — второй тост был бы дублем.
		} finally {
			setUploadTarget(null)
		}
	}

	const action = unit.isKit && canWrite ? (
		<Button
			size='small'
			variant='contained'
			disableElevation
			startIcon={<PlusIcon sx={{ fontSize: 14 }} />}
			onClick={() => setCreateOpen(true)}
			sx={{
				borderRadius: '10px',
				textTransform: 'none',
				fontSize: 12,
				fontWeight: 600,
				px: 1.5,
				py: 0.5,
				minWidth: 0,
			}}
		>
			Сформировать акт
		</Button>
	) : undefined

	return (
		<>
			<DetailCard title='Акты по оборудованию' action={action}>
				{isLoading ? (
					<Loader />
				) : isError || acts.length === 0 ? (
					<EmptyState
						icon={FileSignatureIcon}
						title='Актов по этому оборудованию нет'
						hint={
							unit.isKit
								? 'Сформируйте акт передачи — он появится здесь'
								: 'Акты появятся, когда единицу включат в выдачу'
						}
					/>
				) : (
					<Stack spacing={1}>
						{acts.map(act => (
							<ActRow
								key={act.id}
								act={act}
								canWrite={canWrite}
								onOpen={setDetail}
								onDownload={a => (a.status === 'signed' ? downloadSigned(a) : downloadDocument(a))}
								onUpload={pickSignedFile}
							/>
						))}
					</Stack>
				)}
			</DetailCard>

			<input
				ref={fileInputRef}
				type='file'
				accept='image/*,application/pdf'
				hidden
				onChange={handleSignedFileChange}
			/>

			{createOpen && <CreateActDialog open unit={unit} onClose={() => setCreateOpen(false)} />}

			{detail && (
				<ActDetailDialog
					act={detail}
					canWrite={canWrite}
					onClose={() => setDetail(null)}
					onDownloadDocument={downloadDocument}
					onDownloadSigned={downloadSigned}
					onUpload={pickSignedFile}
				/>
			)}
		</>
	)
}

type DetailProps = {
	act: IEquipmentAct
	canWrite: boolean
	onClose: () => void
	onDownloadDocument: (act: IEquipmentAct) => void
	onDownloadSigned: (act: IEquipmentAct) => void
	onUpload: (act: IEquipmentAct) => void
}

// ActDetailDialog — карточка акта: реквизиты, снимок состава и — у подписанного —
// просмотр загруженного скана с возможностью скачать.
const ActDetailDialog: FC<DetailProps> = ({
	act,
	canWrite,
	onClose,
	onDownloadDocument,
	onDownloadSigned,
	onUpload,
}) => {
	const signed = act.status === 'signed'

	return (
		<Dialog open onClose={onClose} fullWidth maxWidth='md'>
			<DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
				<FileTextIcon sx={{ fontSize: 20, color: '#2563eb', flexShrink: 0 }} />
				{/* minWidth: 0 + overflow/ellipsis: без них flex-элемент не сжимается
				    ниже ширины текста (min-width: auto), строка переполняется, и чип
				    накладывается на номер — «АКТ-…» превращается в многоточие. */}
				<Typography
					sx={{
						fontSize: 16,
						fontWeight: 600,
						color: '#0f172a',
						flex: 1,
						minWidth: 0,
						overflow: 'hidden',
						textOverflow: 'ellipsis',
						whiteSpace: 'nowrap',
					}}
				>
					{act.number}
				</Typography>
				<ActStatusChip status={act.status} />
				<IconButton onClick={onClose} sx={{ ml: 0.5 }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ bgcolor: '#fff' }}>
				<Stack spacing={2}>
					<Box
						sx={{
							display: 'grid',
							gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' },
							gap: 2,
							p: { xs: 1.5, sm: 2 },
							border: '1px solid #e9eef5',
							borderRadius: '12px',
							bgcolor: '#f8fafc',
						}}
					>
						<MetaField label='Дата' value={formatDate(act.issuedAt)} />
						<MetaField label='Вид' value={ACT_KIND_LABEL} />
						<MetaField label='Получатель' value={act.recipientName || '—'} />
					</Box>

					{signed && (
						<Box>
							<Typography sx={{ fontSize: 12, fontWeight: 700, color: '#94a3b8', mb: 1 }}>
								ПОДПИСАННЫЙ АКТ
							</Typography>
							<SignedActPreview act={act} />
						</Box>
					)}

					<Box>
						<Typography sx={{ fontSize: 12, fontWeight: 700, color: '#94a3b8', mb: 1 }}>
							СОСТАВ ({act.content?.length ?? 0})
						</Typography>
						<Box sx={{ border: '1px solid #e9eef5', borderRadius: '12px', overflow: 'hidden' }}>
							<Table size='small'>
								<TableHead>
									<TableRow sx={{ bgcolor: '#fafbfc' }}>
										<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Наименование</TableCell>
										<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Тип</TableCell>
										<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Серийный №</TableCell>
										<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Инв. №</TableCell>
									</TableRow>
								</TableHead>
								<TableBody>
									{(act.content ?? []).map(p => (
										<TableRow key={p.equipmentId}>
											<TableCell>{p.name}</TableCell>
											<TableCell>{p.typeName}</TableCell>
											<TableCell>{p.serialNumber || '—'}</TableCell>
											<TableCell>{p.inventoryNumber || '—'}</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</Box>
					</Box>

					{act.comment && (
						<Box>
							<Typography sx={{ fontSize: 12, fontWeight: 700, color: '#94a3b8', mb: 0.5 }}>
								КОММЕНТАРИЙ
							</Typography>
							<Typography sx={{ fontSize: 14, color: '#334155', whiteSpace: 'pre-wrap' }}>{act.comment}</Typography>
						</Box>
					)}
				</Stack>
			</DialogContent>

			<DialogActions sx={{ px: { xs: 1.5, sm: 3 }, py: 2 }}>
				{canWrite && !signed && (
					<Button
						startIcon={<UploadIcon sx={{ fontSize: 16 }} />}
						onClick={() => {
							onClose()
							onUpload(act)
						}}
						sx={softButtonSx}
					>
						Загрузить подписанный
					</Button>
				)}
				{signed && (
					<Button
						startIcon={<DownloadIcon sx={{ fontSize: 16 }} />}
						onClick={() => onDownloadSigned(act)}
						sx={softButtonSx}
					>
						Скачать подписанный
					</Button>
				)}
				<Button
					variant='contained'
					disableElevation
					startIcon={<DownloadIcon sx={{ fontSize: 16 }} />}
					onClick={() => onDownloadDocument(act)}
					sx={{ borderRadius: '10px', textTransform: 'none', fontSize: 12, fontWeight: 600 }}
				>
					Скачать акт
				</Button>
			</DialogActions>
		</Dialog>
	)
}

// MetaField — пара «подпись/значение» в реквизитах акта.
const MetaField: FC<{ label: string; value: string }> = ({ label, value }) => (
	<Box>
		<Typography sx={{ fontSize: 12, fontWeight: 700, color: '#94a3b8', mb: 0.25 }}>{label}</Typography>
		<Typography sx={{ fontSize: 14, fontWeight: 500, color: '#0f172a' }} noWrap>
			{value}
		</Typography>
	</Box>
)

// SignedActPreview показывает файл подписанного акта: фото с телефона открывается
// картинкой, pdf и прочее — блоком с именем файла. Компонент монтируется только в
// карточке подписанного акта, поэтому запрос не уходит для каждой строки списка.
const SignedActPreview: FC<{ act: IEquipmentAct }> = ({ act }) => {
	const { data, isLoading, isError } = useGetEquipmentActSignedFileQuery(act.id)
	const url = data?.url ?? ''

	const isImage = (act.signedMimeType ?? '').startsWith('image/')

	if (isLoading) return <Loader />
	if (isError || !url) {
		return <Typography sx={{ fontSize: 13, color: '#64748b' }}>Не удалось открыть подписанный акт</Typography>
	}

	if (!isImage) {
		return (
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					gap: 1.5,
					p: 2,
					border: '1px solid #e9eef5',
					borderRadius: '12px',
					bgcolor: '#f8fafc',
				}}
			>
				<FileSignatureIcon sx={{ fontSize: 24, color: '#2563eb' }} />
				<Typography sx={{ fontSize: 14, color: '#0f172a' }}>{act.signedFileName || 'Подписанный акт'}</Typography>
			</Box>
		)
	}

	return (
		<Box
			component='img'
			src={url}
			alt={`Подписанный акт ${act.number}`}
			sx={{
				display: 'block',
				maxWidth: '100%',
				maxHeight: 480,
				mx: 'auto',
				borderRadius: '12px',
				border: '1px solid #e9eef5',
			}}
		/>
	)
}

type CreateProps = {
	open: boolean
	unit: IEquipment
	onClose: () => void
}

// CreateActDialog формирует акт по текущему составу комплекта. Выбор комплекта
// убран намеренно: диалог открывается со вкладки конкретной единицы, а акт на
// чужой комплект формируют с его карточки.
const CreateActDialog: FC<CreateProps> = ({ open, unit, onClose }) => {
	const { data: usersData } = useGetRealmUsersQuery('executors', { skip: !open })
	const { data: sitesData } = useGetAllSitesQuery(undefined, { skip: !open })
	const [create, { isLoading }] = useCreateEquipmentActMutation()

	const [getAct] = useLazyGetEquipmentActQuery()
	const [getDocument] = useLazyGetEquipmentActDocumentQuery()

	const users = usersData?.data ?? []
	const sites = sitesData?.data ?? []

	const [userId, setUserId] = useState<string | null>(null)
	const [siteId, setSiteId] = useState<string | null>(null)
	const [location, setLocation] = useState('')
	const [comment, setComment] = useState('')

	const handleCreate = async () => {
		try {
			const created = await create({
				kitEquipmentId: unit.id,
				recipientUserId: userId,
				siteId,
				location: location || null,
				comment,
			}).unwrap()

			toast.success('Акт сформирован')
			onClose()

			// Акт рождается «сформированным», и его docx сразу отдаём пользователю.
			// Номер нужен для имени файла, но в ответе на создание его нет — берём акт.
			const act = await getAct(created.id).unwrap()
			const blob = await getDocument(created.id).unwrap()
			saveAs(blob, `${act.data.number}.docx`)
		} catch {
			// Ошибку создания показывает onQueryStarted мутации; сбой выгрузки docx —
			// не повод считать акт несозданным, он уже в списке и скачивается оттуда.
		}
	}

	return (
		<Dialog open={open} onClose={onClose} fullWidth maxWidth='sm'>
			<DialogTitle>Формирование акта</DialogTitle>
			<DialogContent dividers>
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
					<Typography sx={{ fontSize: 13, color: '#64748b' }}>
						Комплект: <b>{unit.name}</b>
					</Typography>
					<Autocomplete
						options={users}
						getOptionLabel={customerLabel}
						value={users.find(u => u.id === userId) ?? null}
						onChange={(_e, value) => setUserId(value?.id ?? null)}
						renderInput={params => <TextField {...params} size='small' label='Получатель' />}
					/>
					<Autocomplete
						options={sites}
						getOptionLabel={s => s.name}
						value={sites.find(s => s.id === siteId) ?? null}
						onChange={(_e, value) => setSiteId(value?.id ?? null)}
						renderInput={params => <TextField {...params} size='small' label='Площадка' />}
					/>
					<TextField
						size='small'
						label='Локация'
						value={location}
						onChange={e => setLocation(e.target.value)}
					/>
					<TextField
						size='small'
						label='Комментарий'
						value={comment}
						onChange={e => setComment(e.target.value)}
						multiline
						minRows={2}
					/>
				</Box>
			</DialogContent>
			<DialogActions>
				<Button onClick={onClose} sx={{ textTransform: 'none' }}>
					Отмена
				</Button>
				<Button onClick={handleCreate} variant='contained' disabled={isLoading} sx={{ textTransform: 'none' }}>
					Сформировать
				</Button>
			</DialogActions>
		</Dialog>
	)
}
