import { type FC } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Checkbox,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormControlLabel,
	IconButton,
	MenuItem,
	TextField,
	Typography,
} from '@mui/material'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IAssignEquipmentDTO, IEquipment } from '../types/equipment'
import { useAssignEquipmentMutation } from '../equipmentApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'

type Props = {
	open: boolean
	onClose: () => void
	unit: IEquipment
}

const fieldLabelSx = { fontWeight: 600, mb: 0.5, display: 'block' }
const inputProps = { size: 'small' as const, fullWidth: true }

const userLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full ? `${full} (${u.username})` : u.username
}

export const EquipmentAssignDialog: FC<Props> = ({ open, onClose, unit }) => {
	const [assign, { isLoading }] = useAssignEquipmentMutation()
	const { data: sitesData } = useGetAllSitesQuery()
	const { data: usersData } = useGetRealmUsersQuery('executors')

	const sites = sitesData?.data ?? []
	const users = usersData?.data ?? []

	const { control, handleSubmit } = useForm<IAssignEquipmentDTO>({
		defaultValues: {
			userId: null,
			holderName: '',
			siteId: null,
			location: '',
			comment: '',
			force: false,
			skipAct: false,
		},
		mode: 'onTouched',
	})

	const userId = useWatch({ control, name: 'userId' })
	const holderName = useWatch({ control, name: 'holderName' })
	const siteId = useWatch({ control, name: 'siteId' })
	const location = useWatch({ control, name: 'location' })
	const canSubmit = Boolean(userId || holderName || siteId || location)

	const saveHandler = handleSubmit(async form => {
		const payload: IAssignEquipmentDTO = {
			...form,
			userId: form.userId || null,
			holderName: form.holderName || null,
			siteId: form.siteId || null,
			location: form.location || null,
		}
		try {
			await assign({ id: unit.id, body: payload }).unwrap()
			toast.success('Оборудование выдано')
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при выдаче', { autoClose: false })
		}
	})

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{ paper: { sx: { borderRadius: '16px', p: 1 } } }}
		>
			<DialogTitle
				sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
			>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Выдача оборудования
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent
					dividers
					sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}
				>
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						<Typography variant='body2' sx={{ color: '#6b7280' }}>
							{unit.name}
							{unit.isKit ? ' (комплект)' : ''}
						</Typography>

						<Box>
							<Typography variant='caption' sx={fieldLabelSx}>
								Держатель (сотрудник)
							</Typography>
							<Controller
								control={control}
								name='userId'
								render={({ field }) => (
									<Autocomplete
										options={users}
										getOptionLabel={userLabel}
										value={users.find(u => u.id === field.value) ?? null}
										onChange={(_e, value) => field.onChange(value?.id ?? null)}
										renderInput={params => (
											<TextField {...params} size='small' placeholder='Выберите сотрудника' />
										)}
									/>
								)}
							/>
						</Box>

						<Box>
							<Typography variant='caption' sx={fieldLabelSx}>
								Держатель (название)
							</Typography>
							<Controller
								control={control}
								name='holderName'
								render={({ field }) => (
									<TextField {...field} {...inputProps} placeholder='Или свободный текст' />
								)}
							/>
						</Box>

						<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Площадка
								</Typography>
								<Controller
									control={control}
									name='siteId'
									render={({ field }) => (
										<TextField {...field} select {...inputProps} value={field.value ?? ''}>
											<MenuItem value=''>Не указана</MenuItem>
											{sites.map(s => (
												<MenuItem key={s.id} value={s.id}>
													{s.name}
												</MenuItem>
											))}
										</TextField>
									)}
								/>
							</Box>
							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Локация
								</Typography>
								<Controller
									control={control}
									name='location'
									render={({ field }) => (
										<TextField {...field} {...inputProps} placeholder='Склад, полка A' />
									)}
								/>
							</Box>
						</Box>

						<Box>
							<Typography variant='caption' sx={fieldLabelSx}>
								Комментарий
							</Typography>
							<Controller
								control={control}
								name='comment'
								render={({ field }) => <TextField {...field} multiline minRows={2} {...inputProps} />}
							/>
						</Box>

						<Controller
							control={control}
							name='force'
							render={({ field }) => (
								<FormControlLabel
									control={
										<Checkbox
											checked={Boolean(field.value)}
											onChange={e => field.onChange(e.target.checked)}
										/>
									}
									label='Переназначить принудительно (у состава может быть другой держатель)'
								/>
							)}
						/>

						<Controller
							control={control}
							name='skipAct'
							render={({ field }) => (
								<FormControlLabel
									control={
										<Checkbox
											checked={Boolean(field.value)}
											onChange={e => field.onChange(e.target.checked)}
										/>
									}
									label='Не формировать акт выдачи'
								/>
							)}
						/>
					</Box>
				</DialogContent>

				<DialogActions sx={{ p: 2, gap: 1, justifyContent: 'flex-end' }}>
					<Button
						type='button'
						onClick={onClose}
						variant='outlined'
						sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
					>
						Отмена
					</Button>
					<Button
						type='submit'
						variant='contained'
						disabled={isLoading || !canSubmit}
						sx={{ textTransform: 'none', px: 3 }}
					>
						Выдать
					</Button>
				</DialogActions>
			</form>
		</Dialog>
	)
}
