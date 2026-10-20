import type { FC } from 'react'
import type { SxProps, Theme } from '@mui/material'

import { resolveTypeIcon } from '../typeIcons'

type IconProps = {
	isKit?: boolean
	// icon — сохранённый ключ иконки типа ('' = «Авто»), typeName — название
	// типа, по которому «Авто» подбирает иконку (см. resolveTypeIcon).
	icon?: string
	typeName?: string
	sx?: SxProps<Theme>
}

// Иконка единицы. Цвет берётся из реестра typeIcons, снаружи через sx
// обычно задаётся только размер.
export const EquipmentTypeIcon: FC<IconProps> = ({ isKit, icon, typeName, sx }) => {
	const { Icon, color } = resolveTypeIcon({ isKit, icon, typeName })
	return <Icon sx={[{ color }, sx]} />
}
