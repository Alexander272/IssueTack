package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// TicketAccessChecker реализует модель доступа к тикетам (read/write/delete/work-доступ).
type TicketAccessChecker interface {
	// CheckAccess проверяет право пользователя на заданное действие над тикетом
	// (read/write/delete) согласно модели доступа: решение принимают атрибуты тикета,
	// а отказ по ним компенсирует IsRealmSupervisor. Realm-wide ticket:read/write
	// на результат не влияют.
	CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	// CheckWorkAccess проверяет право на "рабочий" доступ к тикету:
	// write-доступ или пользователь является исполнителем (assignee).
	CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	// CheckInternalAssigneeAccess проверяет, что пользователь является исполнителем (assignee)
	// тикета или менеджером его группы.
	CheckInternalAssigneeAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	// CheckAccessOnTicket выполняет проверку доступа по уже загруженному тикету —
	// общая основа для CheckAccess и логики статусов. Атрибуты тикета решают всегда,
	// а отказ по ним компенсирует только IsRealmSupervisor; realm-wide
	// ticket:read/write в проверке не участвуют.
	//
	// Реалм берётся из самого тикета (ticket.RealmID), а не извне: тикет грузится
	// по id без фильтра по реалму, поэтому клиентский «realm» из заголовка в решении
	// о доступе означал бы, что supervisor одного реалма проходит проверку на тикет
	// другого (см. ticketRealm).
	CheckAccessOnTicket(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string) error
	// IsRealmSupervisor определяет, является ли пользователь «начальником области» в реалме:
	// ему выданы realm-wide пермишены управления областью (category:write или site:write).
	// Единственная точка владения этой проверкой — через policies.enforcer.
	IsRealmSupervisor(ctx context.Context, userID uuid.UUID, realm string) (bool, error)
	// CanManage проверяет, может ли пользователь «управлять» тикетом: является ли он
	// начальником области (realm supervisor) или менеджером группы, к которой относится
	// тикет. Используется для админских операций поверх тикета (подписки и т.п.).
	CanManage(ctx context.Context, userID uuid.UUID, ticket *models.Ticket) (bool, error)
	// CanEditSubtask проверяет, может ли пользователь править содержимое подзадачи
	// (все поля кроме status): автор подзадачи (created_by) или «управление» тикетом
	// (CanManage — менеджер группы или realm supervisor). Смена статуса подзадачи этим
	// методом не гейтится — она остаётся за work-доступом (CheckWorkAccess на тикете).
	CanEditSubtask(ctx context.Context, userID uuid.UUID, subtask *models.Subtask) (bool, error)
	// CanCreateSubtask проверяет, может ли пользователь создавать подзадачи в тикете:
	// создатель тикета, исполнитель (assignee) или «управление» тикетом (CanManage —
	// менеджер группы / realm supervisor). В отличие от CheckWorkAccess, realm-wide
	// политика ticket:write сама по себе права на создание подзадач не даёт: иначе
	// пользователь мог бы создать подзадачу, но не отредактировать/удалить её.
	// Реалм не передаётся: он берётся из загруженного тикета (см. CanManage).
	CanCreateSubtask(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID) (bool, error)
	// CanCreateTicket проверяет наличие realm-wide write-политики на ресурсе тикетов —
	// достаточно ли прав создать заявку напрямую (без ограничений «рабочего» режима).
	// Реалм здесь берётся извне (тикета ещё нет): корректность значения обеспечивает
	// проверка членства в реалме в middleware (см. middleware.CheckPermissions).
	CanCreateTicket(ctx context.Context, userID uuid.UUID, realm string) (bool, error)
	// CanAdministerTickets проверяет право на операции «поверх» тикета, доступные
	// только начальнику области (realm supervisor): смена группы заявки и передача
	// исполнителя мимо менеджера группы. Намеренно не опирается на ticket:write —
	// realm-wide право работать с заявками есть у рядовых пользователей, и по нему
	// они не должны ни переносить заявку в другую группу, ни переназначать чужую.
	// Реалм берётся из тикета, а не извне — оба вызывающих места тикет уже загрузили.
	CanAdministerTickets(ctx context.Context, userID uuid.UUID, ticket *models.Ticket) (bool, error)
	// ownerReadOnly сообщает, что доступ пользователя к заявке держится только на
	// владении (заказчик): он не автор, не исполнитель, не участник/менеджер группы
	// и не начальник области. Нужен, чтобы заказчик не получал «взять в работу».
	ownerReadOnly(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) (bool, error)
	// IsCustomer сообщает, что пользователь в области тикета — «заказчик», а не
	// «рабочий»: у него нет ни управляемых, ни членских групп, и он не начальник
	// области. Такой пользователь не становится рабочим даже будучи автором заявки —
	// его роль ограничена просмотром, внешними комментариями и решениями владельца.
	IsCustomer(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) (bool, error)
	// CanAttachToTicket проверяет право прикрепить файл к заявке: «рабочий» доступ
	// (CheckWorkAccess) или роль заказчика — автор-без-групп/владелец активной заявки.
	// Вложение к своей заявке — часть сценария заказчика, поэтому здесь он не отсекается,
	// в отличие от подзадач и внутренних комментариев. Замороженные заявки запрещены.
	CanAttachToTicket(ctx context.Context, dto *models.AccessCheckDTO) error
}

// TicketAccessService реализует проверку прав доступа к тикетам.
type TicketAccessService struct {
	repo     repository.Tickets
	groups   Groups
	policies AccessPolicies
}

// NewTicketAccessService создаёт TicketAccessService.
func NewTicketAccessService(repo repository.Tickets, groups Groups, policies AccessPolicies) *TicketAccessService {
	return &TicketAccessService{
		repo:     repo,
		groups:   groups,
		policies: policies,
	}
}

// CheckAccessOnTicket — сердце модели доступа к тикету, работает по уже загруженному тикету
// (общая основа для CheckAccess и проверок способностей в TicketService.Update).
//
// Правило: атрибуты тикета решают всегда, а отказ по ним компенсирует только IsRealmSupervisor —
// «начальник области видит всё». Realm-wide ticket:read/write в этой проверке не участвуют.
//
// Так выглядит заявленная модель доступа:
//
//   - участник группы тикета сохраняет доступ, даже если у него нет realm-wide ticket:read —
//     атрибуты не зависят от coarse-прав;
//   - начальник области (IsRealmSupervisor) обходит атрибуты, поэтому зрительная модель
//     «супервизор видит всё» одинакова в CheckAccessOnTicket, CanManage, canChangeStatus,
//     isAdmin и во фронтенде (capabilities.isRealmAdmin).
//
// Раньше здесь стоял Enforce(user, realm, ticket, action), но обе его ветки сводились к одному
// и тому же вызову checkTicketAttributes, поэтому результат проверки ни на что не влиял: доступ
// определялся формулой «атрибуты ИЛИ (супервизор И coarse)». Право ticket:write есть у рядовых
// пользователей, а CanManage/canChangeStatus смотрели только на супервизора, из-за чего
// supervisor без coarse-прав получал CanManage=true, но CheckAccess(Write) отказывал —
// фронтенд показывал кнопки, которых бэкенд не пропускал. Отказ по Casbin убран целиком:
// coarse-права на тикет ничего не решают, поэтому платить за них проверкой не нужно.
//
// Атрибутная модель (первый шаг; поверх неё для любого действия может пройти
// начальник области — см. ниже):
//   - Read: создатель, исполнитель, заказчик (owner), участник группы тикета или её менеджер;
//   - Write: создатель тикета или менеджер группы;
//   - Delete: атрибутно только менеджер группы (создатель удалять не может),
//     но supervisor обходит это так же, как Read/Write.
//
// Менеджер группы определяется перебором GetManagedGroups, поэтому на одно действие
// управляемые группы запрашиваются один раз. Если ни одно из правил не сработало — отказ.
//
// Реалм для обхода «видно всё» берётся из ticket.RealmID, а не из аргумента вызывающего.
// Тикет загружается по id без фильтра по реалму (TicketRepo.GetByID → WHERE t.id = $1),
// поэтому realm, пришедший от клиента, не имеет отношения к тому, чей это тикет: supervisor
// реалма A подставил бы свой realm и получил доступ к тикету реалма B. Аргумент realm
// поэтому убран из сигнатуры, а не просто проигнорирован.
func (s *TicketAccessService) CheckAccessOnTicket(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string) error {
	if err := s.checkTicketAttributes(ctx, ticket, userID, action); err == nil {
		return nil
	}

	// Обход «видно всё» для начальника области. Проверяется только на пути отказа, чтобы
	// горячий путь «своя заявка» не платил лишние Enforce за category/site.
	supervisor, err := s.IsRealmSupervisor(ctx, userID, ticketRealm(ticket))
	if err != nil {
		return err
	}
	if supervisor {
		return nil
	}

	return models.ErrPermissionDenied
}

// ticketRealm возвращает реалм тикета строкой для Casbin-домена. Тикет без реалма
// (внегрупповая заявка) даёт пустой домен — как и раньше в CanManage.
func ticketRealm(ticket *models.Ticket) string {
	if ticket.RealmID == nil {
		return ""
	}
	return ticket.RealmID.String()
}

// checkTicketAttributes применяет модель доступа по атрибутам тикета: ни realm-wide
// ticket:read/write, ни роль в области на неё не влияют.
func (s *TicketAccessService) checkTicketAttributes(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string) error {
	isCreator := ticket.Creator.ID == userID
	isAssignee := ticket.Assignee != nil && ticket.Assignee.ID == userID
	// Заказчик (owner) — полноправный участник заявки: он видит свою заявку и может
	// комментировать её/принимать решения. Чтение owner'у даётся здесь, а вот Write
	// и Delete — нет: иначе owner прошёл бы общие проверки прав и смог бы выставлять
	// произвольные статусы в обход строгих переходов ownerTransitionAllowed (см.
	// TicketService.Update). Право на комментарий owner'у выдаётся отдельно в
	// CommentService.Create, чтобы не расширять CheckWorkAccess.
	isOwner := ticket.Owner != nil && ticket.Owner.ID == userID

	// Тикет без группы: группа не может ни расширить, ни сузить доступ, поэтому
	// решение принимается по автору, исполнителю и заказчику. Создателю доступны и
	// чтение, и правка — иначе он не смог бы отредактировать собственную
	// внегрупповую заявку; заказчику — только чтение. Автор-«заказчик» (без групп)
	// правку не получает: его роль в work-модели — не рабочая (см. IsCustomer).
	if ticket.Group == nil {
		switch action {
		case string(access.Read):
			if isCreator || isAssignee || isOwner {
				return nil
			}
		case string(access.Write):
			if isCreator {
				customer, err := s.IsCustomer(ctx, ticket, userID)
				if err != nil {
					return err
				}
				if !customer {
					return nil
				}
			}
			if isAssignee {
				return nil
			}
		}
		return models.ErrPermissionDenied
	}

	switch action {
	case string(access.Read):
		if isCreator || isAssignee || isOwner {
			return nil
		}
		isMember, err := s.groups.IsMember(ctx, ticket.Group.ID, userID)
		if err != nil {
			return fmt.Errorf("failed to check membership: %w", err)
		}
		if isMember {
			return nil
		}
		return s.checkGroupManager(ctx, ticket.Group.ID, userID)
	case string(access.Write):
		if isCreator {
			customer, err := s.IsCustomer(ctx, ticket, userID)
			if err != nil {
				return err
			}
			if !customer {
				return nil
			}
		}
		return s.checkGroupManager(ctx, ticket.Group.ID, userID)
	case string(access.Delete):
		return s.checkGroupManager(ctx, ticket.Group.ID, userID)
	}
	return models.ErrPermissionDenied
}

// checkGroupManager возвращает nil, если пользователь управляет группой тикета.
func (s *TicketAccessService) checkGroupManager(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) error {
	managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
	if err != nil {
		return fmt.Errorf("failed to check managed groups: %w", err)
	}
	for _, gid := range managed {
		if gid == groupID {
			return nil
		}
	}
	return models.ErrPermissionDenied
}

// ownerReadOnly сообщает, что доступ пользователя к заявке держится только на
// владении (он заказчик): пользователь не автор, не исполнитель, не участник и
// не менеджер группы, не начальник области. Заказчику доступны просмотр,
// комментарии и решения владельца, но не «взять в работу» — работа не его роль.
// Read-доступ при этом owner'у уже выдан в checkTicketAttributes, а вот Take
// гейтится этим методом: иначе заказчик подхватывал бы чужую заявку в исполнители.
func (s *TicketAccessService) ownerReadOnly(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) (bool, error) {
	if ticket.Owner == nil || ticket.Owner.ID != userID {
		return false, nil
	}
	if ticket.Creator.ID == userID {
		return false, nil
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == userID {
		return false, nil
	}
	if ticket.Group != nil {
		member, err := s.groups.IsMember(ctx, ticket.Group.ID, userID)
		if err != nil {
			return false, fmt.Errorf("failed to check membership: %w", err)
		}
		if member || s.checkGroupManager(ctx, ticket.Group.ID, userID) == nil {
			return false, nil
		}
	}
	supervisor, err := s.IsRealmSupervisor(ctx, userID, ticketRealm(ticket))
	if err != nil {
		return false, err
	}
	return !supervisor, nil
}

// IsCustomer сообщает, что пользователь в области тикета — «заказчик»: у него нет
// ни управляемых, ни членских групп в этой области, и он не начальник области.
// Такой пользователь не является «рабочим» даже если он автор заявки: создатель
// с группами — сотрудник, а безгрупповой создатель — внешний заказчик, чья роль
// ограничена просмотром, внешними комментариями и решениями владельца. Именно
// поэтому checkTicketAttributes не выдаёт заказчику Write (см. там), а «взять в
// работу»/подзадачи/внутренние комментарии для него отсекаются производно.
func (s *TicketAccessService) IsCustomer(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) (bool, error) {
	supervisor, err := s.IsRealmSupervisor(ctx, userID, ticketRealm(ticket))
	if err != nil {
		return false, err
	}
	if supervisor {
		return false, nil
	}

	managed, err := s.groups.GetManagedGroups(ctx, userID, ticket.RealmID)
	if err != nil {
		return false, fmt.Errorf("failed to get managed groups: %w", err)
	}
	if len(managed) > 0 {
		return false, nil
	}

	member, err := s.groups.GetMemberGroups(ctx, userID, ticket.RealmID)
	if err != nil {
		return false, fmt.Errorf("failed to get member groups: %w", err)
	}
	return len(member) == 0, nil
}

// CanAttachToTicket проверяет право прикрепить файл к заявке. Кроме «рабочего»
// доступа (CheckWorkAccess) вложение разрешено заказчику — автору-без-групп или
// владельцу: приложить файл к своей активной заявке — часть сценария заказчика,
// поэтому здесь он не отсекается, в отличие от подзадач и внутренних комментариев.
// Замороженные заявки (resolved/closed/cancelled) закрыты и для заказчика.
func (s *TicketAccessService) CanAttachToTicket(ctx context.Context, dto *models.AccessCheckDTO) error {
	workErr := s.CheckWorkAccess(ctx, dto)
	if workErr == nil {
		return nil
	}

	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket: %w", err)
	}
	if isTicketInactive(ticket.Status) {
		return models.ErrTicketFrozen
	}

	isCreator := ticket.Creator.ID == dto.UserID
	isOwner := ticket.Owner != nil && ticket.Owner.ID == dto.UserID
	if !isCreator && !isOwner {
		return workErr
	}
	return nil
}

// CheckAccess — точка входа проверки доступа по DTO. Загружает тикет и делегирует
// единой логике CheckAccessOnTicket, чтобы вся модель доступа по атрибутам была
// описана в одном месте. Раньше здесь стоял собственный Enforce с коротким замыканием:
// при coarse-разрешении тикет даже не грузился, а при провале шаг вхождения повторялся
// внутри CheckAccessOnTicket. Теперь шаг ровно один.
func (s *TicketAccessService) CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket for access check: %w", err)
	}
	return s.CheckAccessOnTicket(ctx, ticket, dto.UserID, dto.Action)
}

// CheckWorkAccess — "рабочий" доступ к тикету: либо write-доступ (по модели CheckAccess),
// либо пользователь является исполнителем. Исполнителю разрешены операции ведения тикета —
// смена статуса, подзадачи, вложения, комментарии — даже без прав создателя/менеджера.
//
// Тикет загружается один раз в начале метода, чтобы:
//   - запретить любые "рабочие" операции (комментарии, вложения, подзадачи) для заявок
//     в неактивном статусе (resolved/closed/cancelled) — независимо от прав пользователя;
//   - затем проверить доступ по атрибутам загруженного тикета (write или исполнитель),
//     не выполняя двойную загрузку.
func (s *TicketAccessService) CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket: %w", err)
	}

	if isTicketInactive(ticket.Status) {
		return models.ErrTicketFrozen
	}

	if err := s.CheckAccessOnTicket(ctx, ticket, dto.UserID, string(access.Write)); err == nil {
		return nil
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
		return nil
	}
	return models.ErrPermissionDenied
}

// CheckInternalAssigneeAccess — узкая "ролевая" проверка без Casbin-домена (realm)
// и без учёта создателя: пользователь должен быть исполнителем (assignee) тикета или
// менеджером его группы. Используется там, где важно именно членство роли, а не write-права —
// например, чтобы решить, показывать ли пользователю внутренние комментарии
// (см. CommentService.GetByTicket).
func (s *TicketAccessService) CheckInternalAssigneeAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket: %w", err)
	}

	if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
		return nil
	}

	if ticket.Group != nil {
		managed, err := s.groups.GetManagedGroups(ctx, dto.UserID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	}

	return models.ErrPermissionDenied
}

// IsRealmSupervisor определяет, является ли пользователь «начальником области» в реалме:
// ему выданы realm-wide пермишены управления областью (category:write или site:write).
// Это ролево-настраиваемый критерий (через выдачу прав ролям в БД), без хардкода конкретных ролей.
func (s *TicketAccessService) IsRealmSupervisor(ctx context.Context, userID uuid.UUID, realm string) (bool, error) {
	ok, err := s.policies.Enforce(userID.String(), realm, string(access.ResourceCategory), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return true, nil
	}
	ok, err = s.policies.Enforce(userID.String(), realm, string(access.ResourceSite), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	return ok, nil
}

// CanManage проверяет, может ли пользователь «управлять» тикетом: начальник области
// (realm supervisor) или менеджер группы, к которой относится тикет.
func (s *TicketAccessService) CanManage(ctx context.Context, userID uuid.UUID, ticket *models.Ticket) (bool, error) {
	supervisor, err := s.IsRealmSupervisor(ctx, userID, ticketRealm(ticket))
	if err != nil {
		return false, err
	}
	if supervisor {
		return true, nil
	}

	if ticket.Group != nil {
		managed, err := s.groups.GetManagedGroups(ctx, userID, ticket.RealmID)
		if err != nil {
			return false, err
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return true, nil
			}
		}
	}
	return false, nil
}

// CanCreateTicket проверяет наличие realm-wide write-политики на ресурс тикетов.
func (s *TicketAccessService) CanCreateTicket(ctx context.Context, userID uuid.UUID, realm string) (bool, error) {
	ok, err := s.policies.Enforce(userID.String(), realm, string(access.ResourceTicket), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	return ok, nil
}

// CanAdministerTickets проверяет, является ли пользователь начальником области тикета.
// Основание то же, что и у CanManage; тикет передаётся, чтобы реалм для Casbin-домена
// был взят из него, а не из клиентского realm-заголовка.
func (s *TicketAccessService) CanAdministerTickets(ctx context.Context, userID uuid.UUID, ticket *models.Ticket) (bool, error) {
	return s.IsRealmSupervisor(ctx, userID, ticketRealm(ticket))
}

// CanCreateSubtask — создание подзадач доступно только пользователям с «атрибутной»
// ролью в тикете: создатель, исполнитель либо «управление» тикетом (CanManage —
// менеджер группы / realm supervisor). Realm-wide политика ticket:write (Casbin) права
// на создание не даёт, чтобы не было разрыва «могу создать, но не могу править/удалить».
// На замороженных заявках (resolved/closed/cancelled) создание запрещено всегда —
// как и в CheckWorkAccess.
func (s *TicketAccessService) CanCreateSubtask(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID) (bool, error) {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return false, fmt.Errorf("failed to load ticket: %w", err)
	}

	if isTicketInactive(ticket.Status) {
		return false, nil
	}
	if ticket.Creator.ID == userID {
		customer, err := s.IsCustomer(ctx, ticket, userID)
		if err != nil {
			return false, err
		}
		if !customer {
			return true, nil
		}
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == userID {
		return true, nil
	}
	return s.CanManage(ctx, userID, ticket)
}

// CanEditSubtask — автор подзадачи может править её содержимое всегда; в противном случае
// нужно «управление» тикетом (CanManage: менеджер группы или realm supervisor).
// Ошибки работы с таблицами (поиск автора, загрузка тикета) не приравниваются к отказу
// в доступе — они возвращаются как ошибки, чтобы отличать неполадки от запрета.
func (s *TicketAccessService) CanEditSubtask(ctx context.Context, userID uuid.UUID, subtask *models.Subtask) (bool, error) {
	if subtask.CreatedBy != nil && *subtask.CreatedBy == userID {
		return true, nil
	}

	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: subtask.TicketID})
	if err != nil {
		return false, fmt.Errorf("failed to load ticket for subtask edit check: %w", err)
	}
	return s.CanManage(ctx, userID, ticket)
}
