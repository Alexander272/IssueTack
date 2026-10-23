import { type ChangeEvent, type FC, useRef, useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Chip,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormControlLabel,
	Checkbox,
	IconButton,
	Stack,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { AlertTriangleIcon, CheckIcon, FileUpIcon, InfoIcon, UploadIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IEquipmentImportResult, IEquipmentImportRow } from '../../types/equipment'
import { useImportEquipmentMutation, useLazyGetEquipmentImportTemplateQuery } from '../../equipmentApiSlice'

type Props = {
	open: boolean
	onClose: () => void
}

// accept — форматы, которые понимает импорт: книга Excel и csv. Расширение .xls
// (старый формат) не поддерживается библиотекой разбора.
const accept = '.xlsx,.xlsm,.csv'

// Диалог импорта оборудования из файла.
//
// Загрузка и разбор — на сервере: клиент только отдаёт файл и показывает отчёт
// по строкам. Ошибка в строке не отменяет импорт остальных, поэтому после
// загрузки показываются и созданные единицы, и причины отказов — по ним файл
// правят и загружают снова.
export const EquipmentImportDialog: FC<Props> = ({ open, onClose }) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))

	const [file, setFile] = useState<File | null>(null)
	const [result, setResult] = useState<IEquipmentImportResult | null>(null)
	// Показывать все строки или только отказы: отказы — то, что нужно чинить,
	// но иногда нужно убедиться, что созданные строки прочитались правильно.
	const [showAll, setShowAll] = useState(false)
	const inputRef = useRef<HTMLInputElement>(null)

	const [importEquipment, { isLoading }] = useImportEquipmentMutation()
	const [getTemplate, { isFetching: isTemplateLoading }] = useLazyGetEquipmentImportTemplateQuery()

	const selectFile = (event: ChangeEvent<HTMLInputElement>) => {
		const selected = event.target.files?.[0] ?? null
		// Сброс значения позволяет выбрать тот же файл повторно (например, после правки).
		event.target.value = ''
		if (!selected) return
		setFile(selected)
		setResult(null)
	}

	const closeHandler = () => {
		setFile(null)
		setResult(null)
		setShowAll(false)
		onClose()
	}

	const submitHandler = async () => {
		if (!file) return
		try {
			const response = await importEquipment(file).unwrap()
			setResult(response.data)
			if (response.data.failed === 0) {
				toast.success(`Импортировано единиц: ${response.data.created}`)
			}
		} catch {
			// Текст ошибки уже показан тостом из слайса: здесь только не даём
			// диалогу закрыться, чтобы файл можно было заменить.
		}
	}

	const rows = result ? (showAll ? result.rows : result.rows.filter(row => row.status === 'failed')) : []

	return (
		<Dialog
			open={open}
			onClose={closeHandler}
			fullWidth
			fullScreen={isMobile}
			maxWidth='md'
			slotProps={{
				paper: {
					sx: {
						borderRadius: { xs: 0, sm: '16px' },
						maxHeight: { xs: 'none', sm: '90vh' },
					},
				},
			}}
		>
			<DialogTitle
				sx={{
					m: 0,
					px: { xs: 1.5, sm: 2 },
					py: 2,
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'flex-start',
				}}
			>
				<Stack>
					<Typography variant='h6' sx={{ fontWeight: 'bold' }}>
						Импорт оборудования
					</Typography>
					<Typography variant='caption' sx={{ color: 'text.secondary' }}>
						Загрузка единиц из файла Excel или CSV
					</Typography>
				</Stack>
				<IconButton onClick={closeHandler} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ px: { xs: 1.5, sm: 3 }, py: 2 }}>
				<Stack spacing={2}>
					{/* Шаблон: без него колонки приходится угадывать, а импорт ищет
					    заголовки по синонимам и требует «Наименование» и «Тип». */}
					<Alert
						severity='info'
						icon={<InfoIcon sx={{ fontSize: 20 }} />}
						sx={{ alignItems: 'center', '& .MuiAlert-message': { width: '100%' } }}
					>
						<Stack
							direction={{ xs: 'column', sm: 'row' }}
							spacing={1}
							sx={{ alignItems: { sm: 'center' }, justifyContent: 'space-between' }}
						>
							<Typography variant='body2'>
								Колонки: наименование и тип обязательны. Получатель ищется среди сотрудников, а если его
								нет — сохраняется свободным значением. Неизвестные колонки станут характеристиками.
							</Typography>
							<Button
								size='small'
								onClick={() => getTemplate()}
								disabled={isTemplateLoading}
								sx={{ textTransform: 'none', flexShrink: 0 }}
							>
								<FileUpIcon sx={{ fontSize: 16, mr: 1 }} />
								Скачать шаблон
							</Button>
						</Stack>
					</Alert>

					{!result && (
						<Box
							sx={{
								display: 'flex',
								flexDirection: { xs: 'column', sm: 'row' },
								alignItems: { sm: 'center' },
								gap: 1.5,
								p: 2,
								borderRadius: '10px',
								border: '1px dashed',
								borderColor: 'grey.400',
								bgcolor: 'grey.50',
							}}
						>
							<Box sx={{ minWidth: 0, flex: 1 }}>
								<Typography sx={{ fontSize: '0.875rem', fontWeight: 600 }}>
									{file ? file.name : 'Файл не выбран'}
								</Typography>
								<Typography variant='caption' sx={{ color: 'text.secondary' }}>
									{file
										? `${(file.size / 1024).toFixed(0)} КБ · ${accept}`
										: 'Поддерживаются .xlsx, .xlsm и .csv'}
								</Typography>
							</Box>
							<input ref={inputRef} type='file' accept={accept} hidden onChange={selectFile} />
							<Button
								variant='outlined'
								color='inherit'
								onClick={() => inputRef.current?.click()}
								sx={{
									textTransform: 'none',
									background: '#fff',
									borderColor: 'grey.400',
									flexShrink: 0,
								}}
							>
								<UploadIcon sx={{ fontSize: 16, mr: 1 }} />
								{file ? 'Заменить файл' : 'Выбрать файл'}
							</Button>
						</Box>
					)}

					{result && (
						<>
							{/* Итог: сколько прочитано, создано и не прошло. Счётчики
							    раздельные — «не прошло» это не то же, что «не прочитано». */}
							<Stack direction='row' spacing={1} sx={{ flexWrap: 'wrap', gap: 1 }}>
								<Chip label={`Строк: ${result.total}`} variant='outlined' />
								<Chip
									icon={<CheckIcon sx={{ fontSize: 14 }} />}
									label={`Создано: ${result.created}`}
									color='success'
									variant='outlined'
								/>
								<Chip
									icon={<AlertTriangleIcon sx={{ fontSize: 14 }} />}
									label={`С ошибкой: ${result.failed}`}
									color={result.failed > 0 ? 'error' : 'default'}
									variant='outlined'
								/>
							</Stack>

							{result.failed === 0 ? (
								<Alert severity='success'>Все строки импортированы.</Alert>
							) : (
								<Alert severity='warning'>
									Строки с ошибкой не созданы — остальные импортированы. Исправьте их в файле и загрузите
									заново.
								</Alert>
							)}

							<FormControlLabel
								control={
									<Checkbox
										size='small'
										checked={showAll}
										onChange={event => setShowAll(event.target.checked)}
									/>
								}
								label={<Typography variant='body2'>Показать все строки</Typography>}
								sx={{ mx: 0 }}
							/>

							<Box
								sx={{
									maxHeight: 320,
									overflowY: 'auto',
									border: '1px solid',
									borderColor: 'grey.200',
									borderRadius: '8px',
								}}
							>
								{rows.length === 0 && (
									<Typography variant='body2' sx={{ p: 2, color: 'text.secondary' }}>
										Строк с ошибками нет.
									</Typography>
								)}
								{rows.map(row => (
									<ImportRow key={`${row.row}-${row.name}`} row={row} />
								))}
							</Box>

							<Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
								<Button
									variant='outlined'
									color='inherit'
									disabled={isLoading}
									onClick={() => {
										// Файл тоже сбрасываем: повторная загрузка того же файла создала бы дубли —
										// импорт не проверяет существование единиц, а создать дубликаты можно.
										setFile(null)
										setResult(null)
										setShowAll(false)
									}}
									sx={{ textTransform: 'none', background: '#fff', borderColor: 'grey.400' }}
								>
									Загрузить ещё файл
								</Button>
								{result.failed > 0 && (
									<Button
										variant='text'
										onClick={() => getTemplate()}
										disabled={isTemplateLoading}
										sx={{ textTransform: 'none' }}
									>
										<FileUpIcon sx={{ fontSize: 16, mr: 1 }} />
										Скачать шаблон
									</Button>
								)}
							</Box>
						</>
					)}
				</Stack>
			</DialogContent>

			<DialogActions
				sx={{
					justifyContent: 'space-between',
					bgcolor: 'grey.50',
					px: { xs: 1.5, sm: 2 },
					py: 2,
					flexDirection: { xs: 'column', sm: 'row' },
					alignItems: { xs: 'stretch', sm: 'center' },
					gap: { xs: 1.5, sm: 0 },
				}}
			>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, color: 'text.secondary' }}>
					<InfoIcon sx={{ fontSize: 16 }} />
					<Typography variant='body2'>Уже заведённые единицы не изменяются</Typography>
				</Box>

				<Box sx={{ display: 'flex', gap: 1, width: { xs: '100%', sm: 'auto' } }}>
					<Button
						type='button'
						onClick={closeHandler}
						variant='outlined'
						sx={{
							textTransform: 'none',
							color: 'text.primary',
							borderColor: 'grey.300',
							backgroundColor: 'white',
							flex: { xs: 1, sm: 'none' },
						}}
					>
						Закрыть
					</Button>
					{!result && (
						<Button
							type='button'
							variant='contained'
							disabled={!file || isLoading}
							onClick={submitHandler}
							sx={{ textTransform: 'none', px: 3, flex: { xs: 1, sm: 'none' } }}
						>
							<UploadIcon sx={{ fontSize: 18, mr: 1 }} />
							{isLoading ? 'Импортируем…' : 'Импортировать'}
						</Button>
					)}
				</Box>
			</DialogActions>
		</Dialog>
	)
}

// ImportRow — одна строка отчёта: номер строки в файле, наименование и что с ней
// стало. Ошибка показывается текстом сервера — она уже человеческая («площадка
// «Склад №5» не найдена»), дублировать её своими формулировками не нужно.
const ImportRow: FC<{ row: IEquipmentImportRow }> = ({ row }) => {
	const created = row.status === 'created'
	return (
		<Box
			sx={{
				display: 'flex',
				alignItems: 'flex-start',
				gap: 1.5,
				px: 1.5,
				py: 1,
				borderBottom: '1px solid',
				borderColor: 'grey.100',
				'&:last-of-type': { borderBottom: 'none' },
			}}
		>
			<Chip
				size='small'
				label={row.row}
				sx={{ minWidth: 44, flexShrink: 0, fontVariantNumeric: 'tabular-nums' }}
			/>
			<Box sx={{ minWidth: 0, flex: 1 }}>
				<Typography sx={{ fontSize: '0.875rem', fontWeight: 600, wordBreak: 'break-word' }}>
					{row.name || 'Без наименования'}
				</Typography>
				{created ? (
					<Typography variant='caption' sx={{ color: 'text.secondary' }}>
						{[
							row.inventoryNumber ? `Инв. №${row.inventoryNumber}` : '',
							row.holder ? (row.holderFound ? `Получатель: ${row.holder}` : `Получатель (свободно): ${row.holder}`) : '',
							row.note,
						]
							.filter(Boolean)
							.join(' · ')}
					</Typography>
				) : (
					<Typography variant='caption' sx={{ color: 'error.main' }}>
						{row.error}
					</Typography>
				)}
			</Box>
			<Box sx={{ flexShrink: 0, pt: 0.25 }}>
				{created ? (
					<CheckIcon sx={{ fontSize: 18, color: 'success.main' }} />
				) : (
					<AlertTriangleIcon sx={{ fontSize: 18, color: 'error.main' }} />
				)}
			</Box>
		</Box>
	)
}
