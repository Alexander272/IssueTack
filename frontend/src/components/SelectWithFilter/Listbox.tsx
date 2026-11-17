import { Box, Divider, Stack } from '@mui/material'
import { CheckIcon } from 'lucide-mui'
import { createContext, forwardRef, useContext } from 'react'
import { Virtuoso } from 'react-virtuoso'

export const ListboxComponent = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLElement>>(
	function ListboxComponent(props, ref) {
		const { children, ...other } = props

		const itemData: React.ReactElement<unknown>[] = []
		;(children as React.ReactElement<unknown>[]).forEach(
			(
				item: React.ReactElement<unknown> & {
					children?: React.ReactElement<unknown>[]
				},
			) => {
				itemData.push(item)
				itemData.push(...(item.children || []))
			},
		)

		const itemCount = itemData.length
		const maxVisibleItems = 8
		const itemSize = 39

		const height = itemCount > maxVisibleItems ? maxVisibleItems * itemSize : itemCount * itemSize + 24

		return (
			<div ref={ref}>
				<OuterElementContext.Provider value={other}>
					<Virtuoso
						style={{ height, width: '100%' }}
						totalCount={itemCount}
						overscan={6}
						components={{
							List: OuterElementType,
						}}
						itemContent={index => <Row data={itemData} index={index} />}
					/>
				</OuterElementContext.Provider>
			</div>
		)
	},
)

type RowProps = {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	data: any[]
	index: number
}

const Row = ({ data, index }: RowProps) => {
	const dataSet = data[index]

	const { key, ...optionProps } = dataSet[0]
	const option = dataSet[1]
	const { selected } = dataSet[2]

	return (
		<li key={key} style={{ width: 'calc(100% - 12px)' }}>
			<Stack width={'100%'} {...optionProps}>
				<Stack direction={'row'} sx={{ width: '100%' }}>
					<Box
						sx={{
							display: 'flex',
							justifyContent: 'center',
							alignItems: 'center',
							width: 20,
							minWidth: 20,
							height: 20,
							mr: 2,
							ml: 1,
							borderRadius: 1,
							border: '1px solid #afafaf',
						}}
					>
						{selected ? <CheckIcon sx={{ fontSize: 14 }} color={'primary'} /> : null}
					</Box>
					<Box
						sx={{
							overflow: 'hidden',
							textOverflow: 'ellipsis',
							whiteSpace: 'nowrap',
						}}
					>
						{option.label}
					</Box>
				</Stack>
				<Divider flexItem sx={{ mt: 1 }} />
			</Stack>
		</li>
	)
}

const OuterElementContext = createContext({})

const OuterElementType = forwardRef<HTMLDivElement>((props, ref) => {
	const outerProps = useContext(OuterElementContext)
	return (
		<div
			ref={ref}
			{...props}
			{...outerProps}
			style={{
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				...(props as any).style,
				overflow: 'hidden', // 🔥 ключевое
			}}
		/>
	)
})
