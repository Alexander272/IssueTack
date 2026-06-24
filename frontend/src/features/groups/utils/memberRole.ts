export function getMemberRoleInfo(
	memberId: string,
	managerId: string | null | undefined,
	defaultAssigneeId: string | null | undefined,
): { label: string; color: string } | null {
	if (memberId === managerId) return { label: 'Руководитель', color: '#3b82f6' }
	if (memberId === defaultAssigneeId) return { label: 'По умолчанию', color: '#10b981' }
	return null
}
