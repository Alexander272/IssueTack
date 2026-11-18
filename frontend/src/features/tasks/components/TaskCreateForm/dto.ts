import type { ICategory } from '@/features/categories/types/category'
import type { FormValues } from './types'
import type { ITaskDTO } from '../../types/task'

// Роль заполняющего форму создания заявки. Набор полей, которые форма показывает
// и отправляет, у ролей разный, поэтому роль задаётся явно, а не выводится заново
// из isManager/isExecutor в каждом поле DTO.
export type CreateRole = 'manager' | 'executor' | 'customer'

// Роль → поля, которыми она вправе распоряжаться. Значения полей, которые форме
// этой роли не показываются, уходят на бэкенд как null/undefined, а не «как есть»:
// черновик хранится на пользователя и переживает смену роли (был исполнителем —
// потерял группы). Бэкенд на чужой dueDate отвечает 403, а на чужого assignee —
// назначает его, поэтому гейт обязателен, а не косметичен.
interface RoleFields {
	ownerId: string | null
	groupId: string | null
	assigneeId: string | null
	dueDate: string | null
	notifierIds?: string[]
	ownerMuted?: boolean
}

const roleFields = (role: CreateRole, data: FormValues, category?: ICategory): RoleFields => {
	switch (role) {
		case 'manager':
			return {
				ownerId: data.ownerId || null,
				groupId: data.groupId || null,
				assigneeId: data.assigneeId || null,
				dueDate: data.dueDate || null,
				notifierIds: data.notifierIds ?? [],
				ownerMuted: Boolean(data.ownerMuted && data.ownerId),
			}
		case 'executor':
			// Группу исполнителю выбирать не дают: поле в форме disabled, а бэкенд
			// сам подставляет группу категории, если группа не передана.
			return {
				ownerId: data.ownerId || null,
				groupId: category?.groupId || null,
				assigneeId: data.assigneeId || null,
				dueDate: data.dueDate || null,
				notifierIds: data.notifierIds ?? [],
			}
		case 'customer':
			// Заявитель без групп: заказчик — он сам, исполнителя и срока у него нет.
			// Пустой список «Уведомлять» не отправляем (undefined) — незачем трогать
			// подписки пустым набором.
			return {
				ownerId: null,
				groupId: category?.groupId || null,
				assigneeId: null,
				dueDate: null,
				notifierIds: data.notifierIds?.length ? data.notifierIds : undefined,
			}
		default:
			throw new Error(`Неизвестная роль формы создания заявки: ${String(role)}`)
	}
}

interface BuildCreateDTOArgs {
	data: FormValues
	category?: ICategory
	role: CreateRole
	realmId: string
	currentUserId: string
}

// buildCreateDTO собирает тело запроса на создание заявки. Живёт отдельно от формы
// (и покрывается здесь же), потому что правило «какие поля от какой роли» — это
// доменное правило, а не деталь разметки.
export const buildCreateDTO = ({ data, category, role, realmId, currentUserId }: BuildCreateDTOArgs): ITaskDTO => {
	const fields = roleFields(role, data, category)

	return {
		id: null,
		title: data.title.trim(),
		description: data.description,
		status: 'open',
		priority: data.priority || category?.priority || 'medium',
		realmId,
		siteId: data.siteId,
		categoryId: data.categoryId,
		creatorId: currentUserId,
		managerId: null,
		closedAt: null,
		subtasks: data.subtasks
			.map(s => ({ title: s.title.trim(), description: s.description.trim() }))
			.filter(s => s.title.length > 0),
		equipmentId: data.equipmentId || null,
		...fields,
	}
}
