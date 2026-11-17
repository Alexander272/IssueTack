import { type FC, useLayoutEffect, useMemo, useRef, useState } from 'react'
import {
	Autocomplete,
	autocompleteClasses,
	type AutocompleteCloseReason,
	Box,
	ClickAwayListener,
	InputAdornment,
	Popper,
	Stack,
	styled,
	type SxProps,
	TextField,
	type Theme,
	Tooltip,
	useTheme,
} from '@mui/material'

import { SearchIcon } from '../Icons/SearchIcon'
import { ListboxComponent } from './Listbox'

export type Option = {
	id: string
	label: string
}

type Props = {
	label?: string
	placeholder?: string
	headerLabel?: string
	values: Option[]
	options: Option[]
	disabled?: boolean
	onChange: (values: Option[]) => void
	isLoading?: boolean
	onFocus?: () => void
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	filterOptions?: (options: any[], { inputValue }: any) => any[]
	sx?: SxProps<Theme>
}

export const SelectWithFilter: FC<Props> = ({
	label,
	placeholder,
	headerLabel,
	values,
	options,
	onChange,
	disabled,
	isLoading,
	onFocus,
	filterOptions,
	sx,
}) => {
	const theme = useTheme()
	const [anchor, setAnchor] = useState<HTMLInputElement | null>(null)
	const [inputValue, setInputValue] = useState('')
	const inputRef = useRef<HTMLInputElement | null>(null)
	const [displayValue, setDisplayValue] = useState('')
	const [extraCount, setExtraCount] = useState(0)

	const openHandler = (e: React.MouseEvent<HTMLElement>) => {
		if (!disabled) {
			setAnchor(e.currentTarget as HTMLInputElement)
		}
	}
	const closeHandler = () => {
		setAnchor(null)
	}

	const sortedOptions = useMemo(() => {
		const selectedSet = new Set(values.map(v => v.id))

		return [...values, ...options.filter(o => !selectedSet.has(o.id))]
	}, [options, values])

	const fullValue = useMemo(() => {
		return values.map(v => v.label).join('; ')
	}, [values])

	useLayoutEffect(() => {
		const input = inputRef.current
		if (!input) {
			setDisplayValue(fullValue)
			setExtraCount(0)
			return
		}

		const compute = () => {
			const el = inputRef.current
			if (!el) return

			const labels = values.map(v => v.label)
			if (labels.length === 0) {
				setDisplayValue('')
				setExtraCount(0)
				return
			}

			const style = window.getComputedStyle(el)
			const canvas = document.createElement('canvas')
			const ctx = canvas.getContext('2d')
			if (!ctx) {
				setDisplayValue(fullValue)
				setExtraCount(0)
				return
			}

			ctx.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`

			const paddingLeft = parseFloat(style.paddingLeft) || 0
			const paddingRight = parseFloat(style.paddingRight) || 0
			const available = el.clientWidth - paddingLeft - paddingRight

			const separator = '; '
			const sepWidth = ctx.measureText(separator).width
			// Запас под бейдж "+99" (ширина текста + боковые паддинги + отступ от инпута)
			const getBadgeReserve = (n: number) => ctx.measureText(`+${n}`).width + 24

			let used = 0
			let visible = 0

			for (let i = 0; i < labels.length; i++) {
				const text = labels[i] ?? ''
				const w = ctx.measureText(text).width + (i > 0 ? sepWidth : 0)
				const remaining = labels.length - i - 1
				const reserve = remaining > 0 ? getBadgeReserve(remaining) : 0

				if (used + w + reserve <= available) {
					used += w
					visible = i + 1
				} else {
					break
				}
			}

			if (visible === 0) {
				// Если даже первый элемент длиннее поля — показываем его (обрежется через ellipsis) + остальные в бейдже
				setDisplayValue(labels[0] ?? '')
				setExtraCount(labels.length - 1)
			} else {
				setDisplayValue(labels.slice(0, visible).join(separator))
				setExtraCount(labels.length - visible)
			}
		}

		compute()
		const ro = new ResizeObserver(compute)
		ro.observe(input)
		return () => ro.disconnect()
	}, [fullValue, values])

	return (
		<Stack sx={{ width: '100%', ...sx }}>
			<Tooltip title={fullValue} placement='top-start' disableInteractive>
				<TextField
					label={label}
					placeholder={placeholder}
					value={displayValue}
					onClick={openHandler}
					disabled={disabled}
					slotProps={{
						input: {
							endAdornment:
								extraCount > 0 ? (
									<InputAdornment position='end' sx={{ ml: 0.5 }}>
										<Box
											component='span'
											sx={t => ({
												display: 'inline-flex',
												alignItems: 'center',
												justifyContent: 'center',
												bgcolor: t.palette.primary.main + '14', // Мягкий акцентный фон (8% прозрачности)
												color: 'primary.main',
												border: '1px solid',
												borderColor: t.palette.primary.main + '33',
												fontSize: 12,
												fontWeight: 600,
												px: 0.75,
												height: 22,
												borderRadius: 1,
												lineHeight: 1,
												userSelect: 'none',
												whiteSpace: 'nowrap',
											})}
										>
											+{extraCount}
										</Box>
									</InputAdornment>
								) : null,
						},
						htmlInput: {
							ref: (node: HTMLInputElement | null) => {
								inputRef.current = node
							},
							readOnly: true,
							sx: {
								cursor: 'pointer',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap',
							},
						},
						...(label && {
							inputLabel: {
								shrink: Boolean(anchor) || Boolean(values.length),
							},
						}),
					}}
				/>
			</Tooltip>

			<Popper
				open={Boolean(anchor)}
				anchorEl={anchor}
				placement='bottom-start'
				sx={{
					width: anchor ? anchor.getBoundingClientRect().width : 'auto',
					border: '1px solid #e1e4e8',
					boxShadow: `0 8px 24px rgba(149, 157, 165, 0.2)`,
					color: '#24292e',
					backgroundColor: '#fff',
					borderRadius: 2,
					zIndex: theme.zIndex.modal + 10,
					fontSize: 13,
				}}
			>
				<ClickAwayListener onClickAway={closeHandler}>
					<div>
						{headerLabel && (
							<Box
								sx={t => ({
									borderBottom: '1px solid #30363d',
									padding: '8px 10px',
									fontWeight: 600,
									...t.applyStyles('light', {
										borderBottom: '1px solid #eaecef',
									}),
								})}
							>
								{headerLabel}
							</Box>
						)}
						<Autocomplete
							open
							multiple
							onClose={(_event: React.ChangeEvent<object>, reason: AutocompleteCloseReason) => {
								if (reason === 'escape') {
									closeHandler()
								}
							}}
							value={values}
							inputValue={inputValue}
							onChange={(event, newValue, reason) => {
								if (
									event.type === 'keydown' &&
									((event as React.KeyboardEvent).key === 'Backspace' ||
										(event as React.KeyboardEvent).key === 'Delete') &&
									reason === 'removeOption'
								) {
									return
								}
								onChange(newValue)
							}}
							onInputChange={(_event, newInputValue, reason) => {
								if (reason !== 'input') return
								setInputValue(newInputValue)
							}}
							onFocus={onFocus}
							filterOptions={filterOptions}
							disableCloseOnSelect
							renderValue={() => null}
							loading={isLoading}
							loadingText='Поиск похожих значений...'
							noOptionsText='Ничего не найдено'
							renderOption={(props, option, state) => [props, option, state] as React.ReactNode}
							options={sortedOptions}
							getOptionLabel={option => option.label}
							renderInput={params => (
								<TextField
									ref={params.slotProps.input.ref}
									autoFocus
									fullWidth
									placeholder='Поиск'
									sx={{ padding: '8px 16px', borderBottom: '1px solid #eaecef' }}
									slotProps={{
										...params.slotProps,
										input: {
											...params.slotProps.input,
											startAdornment: (
												<InputAdornment position='start'>
													<SearchIcon sx={{ fontSize: 16, ml: 1 }} />
												</InputAdornment>
											),
										},
									}}
								/>
							)}
							slotProps={{ listbox: { component: ListboxComponent } }}
							slots={{
								popper: PopperComponent,
							}}
						/>
					</div>
				</ClickAwayListener>
			</Popper>
		</Stack>
	)
}

interface PopperComponentProps {
	anchorEl?: unknown
	disablePortal?: boolean
	open: boolean
}

function PopperComponent(props: PopperComponentProps) {
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	const { disablePortal, anchorEl, open, ...other } = props
	return <StyledAutocompletePopper {...other} />
}

const StyledAutocompletePopper = styled('div')(({ theme }) => ({
	[`& .${autocompleteClasses.paper}`]: {
		boxShadow: 'none',
		color: 'inherit',
		fontSize: 13,
	},
	[`& .${autocompleteClasses.listbox}`]: {
		padding: 0,
		backgroundColor: '#fff',

		[`& .${autocompleteClasses.option}`]: {
			padding: 8,
			paddingBottom: 0,
			margin: 6,
			borderRadius: 8,

			[`&.${autocompleteClasses.focused}, &.${autocompleteClasses.focused}[aria-selected="true"]`]: {
				backgroundColor: theme.palette.action.hover,
			},
		},
		'& ul': {
			padding: 0,
			margin: 0,
		},
	},
	[`&.${autocompleteClasses.popperDisablePortal}`]: {
		position: 'relative',
	},
}))
