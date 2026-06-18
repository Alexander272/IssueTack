import { type FC, useEffect } from 'react'
import { MenuItem, Select, type SelectChangeEvent, type SxProps, type Theme, useTheme } from '@mui/material'

import { useAppDispatch, useAppSelector } from '@/hooks/redux'
import { getUserRealms, setRole } from '@/features/user/userSlice'
// import { setPage } from '@/features/table/tableSlice'
import { getRealm, setRealm } from '../realmSlice'

type Props = {
	sx?: SxProps<Theme>
}

export const ActiveRealm: FC<Props> = ({ sx }) => {
	const { palette } = useTheme()

	const realm = useAppSelector(getRealm)
	const dispatch = useAppDispatch()

	const realms = useAppSelector(getUserRealms)

	useEffect(() => {
		if (!realms) return
		const founded = realms.find(e => e.realmId === realm?.id)
		if (founded) return
		if (realms.length && realms[0].realm) dispatch(setRealm(realms[0].realm))
	}, [realms, dispatch, realm])

	const changeHandler = async (event: SelectChangeEvent) => {
		const value = realms.find(e => e.realmId === event.target.value)
		if (!value) return

		dispatch(setRealm(value.realm!))
		dispatch(setRole(value.role?.name || ''))
		// dispatch(setPage(1))
	}

	if ((realms?.length || 0) < 2) return null
	return (
		<Select
			value={realm?.id || ''}
			onChange={changeHandler}
			// disabled={isFetching}
			sx={{
				color: palette.primary.main,
				fontSize: '1.2rem',
				boxShadow: 'none',
				'.MuiOutlinedInput-notchedOutline': { border: 0 },
				'&.MuiOutlinedInput-root:hover .MuiOutlinedInput-notchedOutline': {
					border: 0,
				},
				'&.MuiOutlinedInput-root.Mui-focused .MuiOutlinedInput-notchedOutline': {
					border: 0,
				},
				'.MuiOutlinedInput-input': { padding: '6.5px 10px' },
				...sx,
			}}
		>
			<MenuItem value='' disabled>
				Выберите область
			</MenuItem>
			{realms.map(item => (
				<MenuItem key={item.id} value={item.realmId}>
					{item.realm?.name || 'No name'}
				</MenuItem>
			))}
		</Select>
	)
}
