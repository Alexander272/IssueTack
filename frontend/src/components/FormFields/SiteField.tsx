import { type FC } from 'react'
import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'

import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'

type SiteOption = { id: string; name: string; address?: string }

type Props = {
	name: string
	rules?: object
}

export const SiteField: FC<Props> = ({ name, rules }) => {
	const { control } = useFormContext()
	const { data: sitesData } = useGetAllSitesQuery()
	const sites = (sitesData?.data ?? []) as SiteOption[]

	return (
		<Controller
			name={name}
			control={control}
			rules={rules}
			render={({ field, fieldState }) => (
				<Autocomplete
					options={sites}
					getOptionLabel={o => o.name}
					getOptionKey={o => o.id}
					value={sites.find(s => s.id === field.value) ?? null}
					onChange={(_e, val) => field.onChange(val?.id ?? '')}
					noOptionsText='Нет площадок'
					renderOption={(props, option) => {
						const { key, ...rest } = props
						return (
							<Box component='li' key={key} {...rest}>
								<Box sx={{ minWidth: 0 }}>
									<Typography sx={{ fontSize: '0.8125rem', fontWeight: 700 }}>
										{option.name}
									</Typography>
									{option.address && (
										<Typography
											sx={{
												fontSize: '0.75rem',
												color: '#4B5563',
												overflow: 'hidden',
												textOverflow: 'ellipsis',
												whiteSpace: 'nowrap',
											}}
										>
											{option.address}
										</Typography>
									)}
								</Box>
							</Box>
						)
					}}
					renderInput={params => (
						<TextField
							{...params}
							size='small'
							error={Boolean(fieldState.error)}
							helperText={fieldState.error?.message}
							placeholder='Выберите площадку'
						/>
					)}
				/>
			)}
		/>
	)
}
