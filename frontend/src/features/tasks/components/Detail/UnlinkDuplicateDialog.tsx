import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import type { DuplicateUnlinkTarget } from '../../hooks/useDuplicateUnlink'

interface Props {
	target: DuplicateUnlinkTarget | null
	loading: boolean
	onConfirm: () => void
	onCancel: () => void
}

export const UnlinkDuplicateDialog = ({ target, loading, onConfirm, onCancel }: Props) => (
	<ConfirmDialog
		open={Boolean(target)}
		title={target?.isSelf ? 'Отвязать дубль?' : `Отвязать дубль №${target?.ticketNumber ?? '—'}?`}
		message='Заявка вернётся в статус, который был до привязки, и снова станет доступна для работы.'
		confirmLabel='Отвязать'
		loading={loading}
		loadingLabel='Отвязываем...'
		onConfirm={onConfirm}
		onCancel={onCancel}
	/>
)
