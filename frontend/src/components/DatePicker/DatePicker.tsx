import { styled } from '@mui/material'
import { PickersTextField } from '@mui/x-date-pickers'

export const DateTextField = styled(PickersTextField)({
	'.MuiPickersOutlinedInput-root': {
		borderRadius: '12px',
	},
	'.MuiPickersSectionList-root': {
		padding: '8px 0',
	},
})
