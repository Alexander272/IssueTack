import { useEffect, type FC } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { Box, Button, Stack, Typography, useTheme } from '@mui/material'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IRolePermissionsGrouped, IRoleWithPermsDTO } from '../../types/role'
import { RoleInfoCard } from './RoleInfoCard'
import { InheritanceCard } from './InheritanceCard'
import { PermissionsCard } from './PermissionsCard'
import { useDeleteRoleMutation, useGetRoleWithPermissionsQuery, useUpdateRoleMutation } from '../../roleApiSlice'
import { Fallback } from '@/components/Fallback/Fallback'
import { SaveIcon } from '@/components/Icons/SaveIcon'
import { TrashBinIcon } from '@/components/Icons/TrashBinIcon'
import { BoxFallback } from '@/components/Fallback/BoxFallback'

const defaultValues = {
	name: '',
	slug: '',
	level: 3,
	description: '',
	inherits: [],
	permissions: [],
	perms: [],
}

export interface IForm extends IRoleWithPermsDTO {
	perms: IRolePermissionsGrouped[]
}

type Props = {
	roleId: string
	onCancel?: () => void
	onSuccess?: () => void
}

export const UpdateRole: FC<Props> = ({ roleId, onCancel, onSuccess }) => {
	const { palette } = useTheme()

	const { data: role, isFetching } = useGetRoleWithPermissionsQuery(roleId, { skip: !roleId })

	const [update, { isLoading: isUpdating }] = useUpdateRoleMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteRoleMutation()

	const methods = useForm<IForm>({ defaultValues })
	const { handleSubmit, reset } = methods

	useEffect(() => {
		reset(role?.data || defaultValues)
	}, [role, reset])

	const onSubmit = handleSubmit(async form => {
		console.log('📦 Отправка формы:', form)

		form.permissions = form.perms.flatMap(p =>
			p.resources.filter(r => r.isAssigned).map(r => r.permissionId as string),
		)

		try {
			await update(form).unwrap()
			toast.success('Роль обновлена')

			if (onSuccess) onSuccess()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data.message, { autoClose: false })
		}
	})

	const deleteHandler = async () => {
		try {
			await remove(roleId).unwrap()
			toast.success('Роль удалена')

			if (onSuccess) onSuccess()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data.message, { autoClose: false })
		}
	}

	const handleCancel = () => {
		if (onCancel) onCancel()
	}

	if (isFetching) return <Fallback />

	return (
		<Box>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
					mb: 3,
					flexWrap: 'wrap',
					gap: 2,
				}}
			>
				<Box width={'100%'}>
					<Typography variant='h5' textAlign={'center'} sx={{ fontWeight: 'bold' }}>
						Изменить роль: <span>{role?.data.name || 'Новая роль'}</span>
					</Typography>
					<Typography variant='body2' textAlign={'center'} color='text.secondary'>
						Управление доступом, наследованием и сортировкой
					</Typography>
				</Box>
			</Box>

			{isUpdating || isDeleting ? <BoxFallback /> : null}

			<Stack component={'form'} onSubmit={onSubmit}>
				<FormProvider {...methods}>
					<RoleInfoCard />

					<InheritanceCard roleId={roleId} />
					<PermissionsCard role={role?.data} />

					<Box
						sx={{
							display: 'flex',
							justifyContent: 'flex-end',
							gap: 1.5,
							pt: 3,
							borderTop: '1px solid #f3f4f6',
						}}
					>
						<Button
							variant='outlined'
							color='inherit'
							onClick={handleCancel}
							sx={{ mr: 'auto', textTransform: 'none' }}
						>
							Отменить
						</Button>

						{role?.data && (
							<Button
								variant='outlined'
								color='error'
								onClick={deleteHandler}
								sx={{ textTransform: 'none' }}
							>
								<TrashBinIcon sx={{ mr: 1, fontSize: 16, fill: palette.error.main }} /> Удалить
							</Button>
						)}
						<Button type='submit' variant='outlined' sx={{ textTransform: 'none' }}>
							<SaveIcon fontSize={16} mr={1} fill={palette.primary.main} /> Сохранить роль
						</Button>
					</Box>
				</FormProvider>
			</Stack>
		</Box>
	)
}
