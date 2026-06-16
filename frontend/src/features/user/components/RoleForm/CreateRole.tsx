import { useEffect, type FC } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { Box, Button, Stack, Typography, useTheme } from '@mui/material'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IRolePermissionsGrouped, IRoleWithPermsDTO } from '../../types/role'
import { useGetPermissionsQuery } from '@/features/access/permApiSlice'
import { useCreateRoleMutation } from '../../roleApiSlice'
import { Fallback } from '@/components/Fallback/Fallback'
import { SaveIcon } from '@/components/Icons/SaveIcon'
import { RoleInfoCard } from './RoleInfoCard'
import { InheritanceCard } from './InheritanceCard'
import { PermissionsCard } from './PermissionsCard'
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
	onCancel?: () => void
	onSuccess?: () => void
}

export const CreateRole: FC<Props> = ({ onCancel, onSuccess }) => {
	const { palette } = useTheme()

	const { data, isFetching } = useGetPermissionsQuery(null)
	const [create, { isLoading }] = useCreateRoleMutation()

	const methods = useForm<IForm>({ defaultValues })
	const { handleSubmit, setValue } = methods

	useEffect(() => {
		if (!data?.data) return

		const initialPerms = data.data.map(row => ({
			group: row.group,
			resources: row.items.map(item => ({
				permissionId: item.id,
				object: item.object,
				action: item.action,
				isAssigned: false,
				isInherited: false,
				status: 'original' as const,
			})),
		}))

		setValue('perms', initialPerms)
	}, [data, setValue])

	const onSubmit = handleSubmit(async form => {
		console.log('📦 Отправка формы:', form)
		form.permissions = form.perms.flatMap(p =>
			p.resources.filter(r => r.isAssigned).map(r => r.permissionId as string),
		)

		try {
			await create(form).unwrap()
			toast.success('Роль создана')

			if (onSuccess) onSuccess()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data.message, { autoClose: false })
		}
	})

	const handleCancel = () => {
		if (onCancel) {
			onCancel()
		}
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
						Создать роль: <span>Новая роль</span>
					</Typography>
					<Typography variant='body2' textAlign={'center'} color='text.secondary'>
						Управление доступом, наследованием и сортировкой
					</Typography>
				</Box>
			</Box>

			{isLoading && <BoxFallback />}

			<Stack component={'form'} onSubmit={onSubmit}>
				<FormProvider {...methods}>
					<RoleInfoCard />

					<InheritanceCard />
					<PermissionsCard />

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

						<Button type='submit' variant='outlined' sx={{ textTransform: 'none' }}>
							<SaveIcon fontSize={16} mr={1} fill={palette.primary.main} /> Сохранить роль
						</Button>
					</Box>
				</FormProvider>
			</Stack>
		</Box>
	)
}
