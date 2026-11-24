package models

import (
	"time"

	json "github.com/goccy/go-json"
	"github.com/google/uuid"
)

type Ticket struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`

	// Статусы и приоритеты
	Status   TicketStatus `json:"status" db:"status"`
	Priority Priority     `json:"priority" db:"priority"`

	TicketNumber *int       `json:"ticketNumber,omitempty"`
	RealmID      *uuid.UUID `json:"realmId,omitempty"`

	Site             *SiteShort             `json:"site"`             // Площадка выполнения
	Category         *CategoryShort         `json:"category"`         // Категория
	CategoryModifier *CategoryModifierShort `json:"categoryModifier"` // Подкатегория (модификатор)

	// Кто участвует
	Creator  UserShort   `json:"creator"`  // Кто фактически создал (может быть Manager)
	Owner    *UserShort  `json:"owner"`    // Для кого выполняется (Клиент/Заявитель)
	Group    *GroupShort `json:"group"`    // На какую группу назначен (напр. IT-отдел)
	Assignee *UserShort  `json:"assignee"` // Конкретный агент-исполнитель
	Manager  *UserShort  `json:"manager"`  // Начальник, контролирующий задачу

	// OwnerMuted отключает уведомления заказчику и скрывает заявку из его списка заявок.
	OwnerMuted bool `json:"ownerMuted" db:"owner_muted"`

	// DuplicateOf — основная заявка, для которой эта заявка помечена дублем.
	// PreviousStatus — статус дубля на момент привязки; нужен, чтобы осознанная
	// отвязка вернула заявку в работу, а не оставила её закрытой навсегда.
	DuplicateOf       *uuid.UUID    `json:"duplicateOf,omitempty" db:"duplicate_of"`
	DuplicateOfNumber *int          `json:"duplicateOfNumber,omitempty" db:"-"`
	PreviousStatus    *TicketStatus `json:"previousStatus,omitempty" db:"previous_status"`

	// Сроки и планирование
	DueDate    *time.Time `json:"dueDate" db:"due_date"`
	ClosedAt   *time.Time `json:"closedAt" db:"closed_at"`
	ResolvedAt *time.Time `json:"resolvedAt" db:"resolved_at"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`

	Subtasks    []*Subtask    `json:"subtasks,omitempty"`
	Attachments []*Attachment `json:"attachments,omitempty"`

	// Notifiers — явный список «Уведомлять», который редактирует пользователь.
	// DuplicateNotifiers — неявные наблюдатели (автор/заказчик дублей этой заявки):
	// они получают доступ и уведомления, но не должны попадать в форму редактирования
	// и не копируются в ticket_notifiers при SyncNotifiers.
	Notifiers          []UserShort     `json:"notifiers,omitempty"`
	DuplicateNotifiers []UserShort     `json:"-"`
	Access             *AccessFlags    `json:"access,omitempty"`
	Equipment          *EquipmentShort `json:"equipment,omitempty"`
}

type AccessFlags struct {
	CanRead       bool `json:"canRead"`
	CanWrite      bool `json:"canWrite"`
	CanDelete     bool `json:"canDelete"`
	CanWork       bool `json:"canWork"`
	CanTake       bool `json:"canTake"`
	CanEditFields bool `json:"canEditFields"`
	IsAdmin       bool `json:"isAdmin"`
	IsManager     bool `json:"isManager"`
	// CanCancel/CanConfirm/CanReopen — «решения владельца»: отмена новой заявки,
	// подтверждение решения и возврат в работу. Считаются одной формулой
	// (TicketService.ownerDecisions), той же, что отдаёт плагин Mattermost
	// (PluginTicketDetail), — чтобы веб и плагин не разъезжались.
	CanCancel  bool `json:"canCancel"`
	CanConfirm bool `json:"canConfirm"`
	CanReopen  bool `json:"canReopen"`
	// CanRequestDueDate — исполнитель может запросить перенос срока (при установленном сроке).
	CanRequestDueDate bool `json:"canRequestDueDate"`
	// CanDecideDueDate — менеджер группы/начальник области может утверждать/отклонять запросы переноса срока.
	CanDecideDueDate bool `json:"canDecideDueDate"`
	// CanPickModifier — исполнитель или начальник области может выбрать подкатегорию
	// на открытой заявке; при создании право то же (prepareCreateModifier), но без
	// привязки к конкретному assignee. Формула повторяет prepareUpdateModifier, чтобы
	// фронт не дублировал серверные правила и не показывал контрол там, где его не пропустят.
	CanPickModifier bool `json:"canPickModifier"`

	// CanCreateSubtask/CanAttachToTicket/CanManage — «тонкие» права, которые фронт
	// раньше вычислял сам по access.isManager/isAssignee/isCreator. Считаются теми же
	// формулами, что и сервисные проверки (TicketAccessService), чтобы кнопки не
	// расходились с тем, что бэкенд действительно пропустит.
	CanCreateSubtask  bool `json:"canCreateSubtask"`
	CanAttachToTicket bool `json:"canAttachToTicket"`
	CanManage         bool `json:"canManage"`

	// CanLinkDuplicate/CanUnlinkDuplicate — работа с дублями: связывать может
	// исполнитель/менеджер (рабочий доступ, не заказчик); отвязывать — тот же круг
	// плюс создатель основной заявки, чтобы можно было исправить ошибочную привязку.
	CanLinkDuplicate   bool `json:"canLinkDuplicate"`
	CanUnlinkDuplicate bool `json:"canUnlinkDuplicate"`

	AllowedStatuses []TicketStatus `json:"allowedStatuses"`
}

type GetTicketByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	Actor   *Actor    `json:"actor"`
	RealmID string    `json:"realmId"`
}

type TicketFilter struct {
	Actor                      *Actor         `json:"actor"`
	Number                     *int           `form:"number" json:"number"`
	RealmID                    *uuid.UUID     `form:"realmId,parser=encoding.TextUnmarshaler" json:"realmId"`
	SiteIDs                    []string       `form:"siteIds" json:"siteIds"`
	Status                     *TicketStatus  `form:"status" json:"status" db:"status" binding:"omitempty,enum"`
	Statuses                   []TicketStatus `form:"statuses" json:"statuses"`
	OwnerID                    *uuid.UUID     `form:"ownerId,parser=encoding.TextUnmarshaler" json:"ownerId" db:"owner_id"`
	AssigneeID                 *uuid.UUID     `form:"assigneeId,parser=encoding.TextUnmarshaler" json:"assigneeId" db:"assignee_id"`
	GroupIDs                   []uuid.UUID    `json:"-"`
	IncludeUngroupedAssignedTo *uuid.UUID     `json:"-"`
	Priorities                 []Priority     `form:"priorities" json:"priorities"`
	Search                     *string        `form:"search" json:"search"`
	DueDateFrom                *time.Time     `form:"dueDateFrom" json:"dueDateFrom" time_format:"2006-01-02"`
	DueDateTo                  *time.Time     `form:"dueDateTo" json:"dueDateTo" time_format:"2006-01-02"`
	Sort                       *string        `form:"sort" json:"sort"`
	Mode                       *string        `form:"mode" json:"mode"`
	CreatorID                  *uuid.UUID     `json:"-"`
	// CreatedOrOwnedBy — внутренний фильтр «автор ИЛИ заказчик» (для Mattermost-плагина):
	// тикеты, где пользователь является creator_id ИЛИ owner_id.
	CreatedOrOwnedBy *uuid.UUID `json:"-"`
	// ReadableBy — внутренний фильтр поиска заявки по номеру (Mattermost-бот):
	// тикеты, доступные пользователю на чтение по атрибутам — автор, заказчик,
	// исполнитель ИЛИ группа тикета входит в GroupIDs. Обычный списочный scope
	// «группы актора + внегрупповые назначенные ему» отсекал создателя, который
	// сам в группу тикета не входит (заявка группы, автор — рядовой заявитель).
	ReadableBy *MyWorkFilter `json:"-"`
	Archived   *bool         `form:"archived" json:"archived"`
	// Duplicates — фильтр заявок-дублей: true — только дубли, false — только не-дубли,
	// nil — все (обычный список). Применяется в архиве, где дубли оказываются после
	// закрытия; в активной вкладке дублей быть не должно.
	Duplicates *bool `form:"duplicates" json:"duplicates"`
	// AllStatuses — отдаёт заявки всех статусов, минуя фильтр вкладки (statusMode):
	// «все заявки по оборудованию» истории единицы включает closed/cancelled, а
	// вкладочный набор («Активные»/«Архив») такого объединения не строит. nil/false —
	// фильтр вкладки работает как раньше; сервис сам проставляет ArchiveWithResolved
	// для режима «Мои задачи», это поле приходит только из запроса (deep-link по оборудованию).
	AllStatuses bool `form:"allStatuses" json:"-"`
	// ArchiveWithResolved — страница «Мои задачи» (mode=assigned): resolved считается
	// архивным статусом. Активная вкладка его не показывает, архивная — включает.
	// Ставится сервисом (зависит от режима страницы, из запроса не приходит).
	ArchiveWithResolved bool          `json:"-"`
	FavoritesByUser     *uuid.UUID    `json:"-"`
	FavoriteType        *FavoriteType `json:"-"`
	MyWork              *MyWorkFilter `json:"-"`
	// Фильтр по дате закрытия (используется, напр. в архиве плагина)
	ClosedAtFrom *time.Time `form:"closedAtFrom" json:"closedAtFrom" time_format:"2006-01-02"`
	ClosedAtTo   *time.Time `form:"closedAtTo"   json:"closedAtTo"   time_format:"2006-01-02"`
	EquipmentID  *uuid.UUID `form:"equipmentId,parser=encoding.TextUnmarshaler" json:"equipmentId"`
	Limit        int        `form:"limit" json:"limit" db:"limit" binding:"omitempty,min=0"`
	Offset       int        `form:"offset" json:"offset" db:"offset" binding:"omitempty,min=0"`
}

// MyWorkFilter описывает фильтр «Мои задачи»: тикеты, где пользователь является
// исполнителем лично (assignee), ИЛИ где группа тикета входит в GroupIDs
// (задачи групп). Используется страницей задач (mode=assigned), чтобы
// исполнитель видел и свои назначения, и задачи своих групп.
type MyWorkFilter struct {
	UserID   uuid.UUID
	GroupIDs []uuid.UUID
}

type TicketDTO struct {
	ID          *uuid.UUID `json:"id" db:"id"`
	Actor       *Actor     `json:"actor"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`

	// Статусы и приоритеты
	Status   TicketStatus `json:"status" db:"status"`
	Priority Priority     `json:"priority" db:"priority"`

	TicketNumber       int        `json:"ticketNumber,omitempty"`
	RealmID            *uuid.UUID `json:"realmId"`
	SiteID             uuid.UUID  `json:"siteId"`                                       // Площадка выполнения
	CategoryID         uuid.UUID  `json:"categoryId"`                                   // Категория
	CategoryModifierID *uuid.UUID `json:"categoryModifierId" db:"category_modifier_id"` // Подкатегория (модификатор)

	// Кто участвует
	CreatorID  uuid.UUID  `json:"creatorId" db:"creator_id"`   // Кто фактически создал (может быть Manager)
	OwnerID    *uuid.UUID `json:"ownerId" db:"owner_id"`       // Для кого выполняется (Клиент/Заявитель)
	GroupID    *uuid.UUID `json:"groupId" db:"group_id"`       // На какую группу назначен (напр. IT-отдел)
	AssigneeID *uuid.UUID `json:"assigneeId" db:"assignee_id"` // Конкретный агент-исполнитель
	ManagerID  *uuid.UUID `json:"managerId" db:"manager_id"`   // Начальник, контролирующий задачу

	// Сроки и планирование
	DueDate    *time.Time `json:"dueDate" db:"due_date"`
	ClosedAt   *time.Time `json:"closedAt" db:"closed_at"`
	ResolvedAt *time.Time `json:"resolvedAt" db:"resolved_at"`

	// Подзадачи, создаваемые вместе с заявкой (используется только при Create;
	// при Update — запланированные значения игнорируются).
	Subtasks []*SubtaskDTO `json:"subtasks,omitempty"`

	EquipmentID *uuid.UUID `json:"equipmentId" db:"equipment_id"`

	// Наблюдатели заявки (список «Уведомлять»): получают уведомления по ключевым
	// событиям как владелец и получают read-доступ к заявке. При Update
	// обрабатывается как полная замена, если поле передано в запросе.
	NotifierIDs []uuid.UUID `json:"notifierIds,omitempty"`

	// OwnerMuted отключает уведомления заказчику и скрывает заявку из его списка заявок.
	OwnerMuted bool `json:"ownerMuted" db:"owner_muted"`

	// Поля, переданные в запросе (заполняется при UnmarshalJSON). Позволяет делать partial update.
	Provided map[string]bool `json:"-"`
}

func (dto *TicketDTO) UnmarshalJSON(data []byte) error {
	type Alias TicketDTO
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(dto)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	dto.Provided = make(map[string]bool, len(raw))
	for key := range raw {
		dto.Provided[key] = true
	}
	return nil
}

func (dto *TicketDTO) HasField(key string) bool {
	return dto.Provided != nil && dto.Provided[key]
}

func (dto *TicketDTO) MarkProvided(key string) {
	if dto.Provided == nil {
		dto.Provided = make(map[string]bool)
	}
	dto.Provided[key] = true
}

type DeleteTicketDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	Actor   *Actor    `json:"actor"`
	RealmID string    `json:"realmId"`
}

// TakeTicketDTO описывает «взятие заявки в работу»: пользователь назначает себя
// исполнителем и переводит заявку в статус in_progress.
type TakeTicketDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	Actor   *Actor    `json:"actor"`
	RealmID string    `json:"realmId"`
}

// TransferTicketDTO описывает передачу заявки от текущего исполнителя другому
// участнику той же группы. Меняется только исполнитель, статус не затрагивается.
type TransferTicketDTO struct {
	ID         *uuid.UUID `json:"id" db:"id"`
	Actor      *Actor     `json:"actor"`
	AssigneeID *uuid.UUID `json:"assigneeId" db:"assignee_id"`
	RealmID    string     `json:"realmId"`
}

// LinkDuplicateDTO — привязка заявки к основной. ID (дубль) берётся из URL,
// PrimaryID — из тела; realm и актор добавляет хендлер.
type LinkDuplicateDTO struct {
	ID        uuid.UUID `json:"-" db:"id"`
	PrimaryID uuid.UUID `json:"primaryId" db:"primary_id"`
	Actor     *Actor    `json:"-"`
	RealmID   string    `json:"-"`
}

// UnlinkDuplicateDTO — осознанная отвязка дубля от основной заявки.
type UnlinkDuplicateDTO struct {
	ID      uuid.UUID `json:"-" db:"id"`
	Actor   *Actor    `json:"-"`
	RealmID string    `json:"-"`
}

// DuplicateLinkResult — результат привязки/отвязки: корень цепочки и путь от
// дубля к корню (включая оба конца), чтобы фронт мог показать, куда именно
// переехала или вернулась заявка.
type DuplicateLinkResult struct {
	RootID           uuid.UUID   `json:"rootId"`
	RootTicketNumber *int        `json:"rootTicketNumber,omitempty"`
	Chain            []uuid.UUID `json:"chain"`
}

// GetDuplicatesDTO — запрос списка дублей со стороны основной заявки.
// Actor обязателен: каждая найденная заявка проходит проверку read-доступа,
// иначе список раскрывал бы чужие дубли из других групп.
type GetDuplicatesDTO struct {
	ID    uuid.UUID `json:"-"`
	Actor *Actor    `json:"-"`
}

// DuplicateRef — компактная карточка связанной заявки на основной. Статус не
// отдаётся намеренно: дубли всегда закрыты, он ничего не говорит пользователю.
// Вместо этого показываем объём обсуждения — комментарии и вложения.
type DuplicateRef struct {
	ID               uuid.UUID `json:"id"`
	TicketNumber     *int      `json:"ticketNumber,omitempty"`
	Title            string    `json:"title"`
	CreatorName      string    `json:"creatorName"`
	AssigneeName     string    `json:"assigneeName"`
	CommentsCount    int       `json:"commentsCount"`
	AttachmentsCount int       `json:"attachmentsCount"`
	CreatedAt        time.Time `json:"createdAt"`
	// CanUnlink считается отдельно для каждого дубля: право отвязать чужую
	// заявку с основной не совпадает с правом отвязать саму её.
	CanUnlink bool `json:"canUnlink"`
}

// AccessCheckDTO — вход проверки доступа к тикету. Реалм намеренно не принимается:
// TicketRepo.GetByID грузит тикет по id без фильтра по реалму, поэтому реалм для
// решения о доступе берётся из самого тикета (TicketAccessService.CheckAccessOnTicket).
// Раньше здесь был Realm из клиентского заголовка, и это позволяло supervisor'у одного
// реалма получить доступ к тикету другого (см. middleware.CheckPermissions).
type AccessCheckDTO struct {
	TicketID uuid.UUID
	UserID   uuid.UUID
	Action   string
}

type FieldChange struct {
	Tag    ActivityType
	OldVal string
	NewVal string
}

func (dto *TicketDTO) GetChanges(old *Ticket) []*FieldChange {
	var changes []*FieldChange

	toStr := func(v any) string {
		return formatFieldValue(v)
	}

	if dto.HasField("title") && dto.Title != old.Title {
		changes = append(changes, &FieldChange{ActionTitleChanged, old.Title, dto.Title})
	}
	if dto.HasField("description") && dto.Description != old.Description {
		changes = append(changes, &FieldChange{ActionDescriptionChanged, old.Description, dto.Description})
	}

	if dto.HasField("status") && dto.Status != old.Status {
		changes = append(changes, &FieldChange{ActionStatusChanged, toStr(old.Status), toStr(dto.Status)})
	}
	if dto.HasField("priority") && dto.Priority != old.Priority {
		changes = append(changes, &FieldChange{ActionPriorityChanged, toStr(old.Priority), toStr(dto.Priority)})
	}

	if dto.HasField("dueDate") {
		if dto.DueDate != nil && (old.DueDate == nil || !dto.DueDate.Equal(*old.DueDate)) {
			changes = append(changes, &FieldChange{ActionDueDateChanged, toStr(old.DueDate), toStr(dto.DueDate)})
		} else if dto.DueDate == nil && old.DueDate != nil {
			changes = append(changes, &FieldChange{ActionDueDateChanged, toStr(old.DueDate), "none"})
		}
	}
	if dto.HasField("closedAt") {
		if dto.ClosedAt != nil && (old.ClosedAt == nil || !dto.ClosedAt.Equal(*old.ClosedAt)) {
			changes = append(changes, &FieldChange{ActionClosed, toStr(old.ClosedAt), toStr(dto.ClosedAt)})
		} else if dto.ClosedAt == nil && old.ClosedAt != nil {
			changes = append(changes, &FieldChange{ActionClosed, toStr(old.ClosedAt), "none"})
		}
	}

	if dto.HasField("siteId") {
		if old.Site != nil && dto.SiteID != old.Site.ID {
			changes = append(changes, &FieldChange{ActionSiteChanged, old.Site.ID.String(), dto.SiteID.String()})
		} else if old.Site == nil {
			changes = append(changes, &FieldChange{ActionSiteChanged, "none", dto.SiteID.String()})
		}
	}
	if dto.HasField("categoryId") {
		if old.Category != nil && dto.CategoryID != old.Category.ID {
			changes = append(changes, &FieldChange{ActionCategoryChanged, old.Category.ID.String(), dto.CategoryID.String()})
		} else if old.Category == nil {
			changes = append(changes, &FieldChange{ActionCategoryChanged, "none", dto.CategoryID.String()})
		}
	}
	if dto.HasField("categoryModifierId") {
		oldVal, newVal := "none", "none"
		if old.CategoryModifier != nil {
			oldVal = old.CategoryModifier.ID.String()
		}
		if dto.CategoryModifierID != nil {
			newVal = dto.CategoryModifierID.String()
		}
		if oldVal != newVal {
			changes = append(changes, &FieldChange{ActionSubcategoryChanged, oldVal, newVal})
		}
	}

	if dto.HasField("groupId") {
		if dto.GroupID != nil && (old.Group == nil || *dto.GroupID != old.Group.ID) {
			oldVal := "none"
			if old.Group != nil {
				oldVal = old.Group.ID.String()
				changes = append(changes, &FieldChange{ActionGroupChanged, oldVal, dto.GroupID.String()})
			} else {
				changes = append(changes, &FieldChange{ActionGroupAssigned, oldVal, dto.GroupID.String()})
			}
		} else if dto.GroupID == nil && old.Group != nil {
			changes = append(changes, &FieldChange{ActionGroupChanged, old.Group.ID.String(), "none"})
		}
	}

	if dto.HasField("managerId") {
		if dto.ManagerID != nil && (old.Manager == nil || *dto.ManagerID != old.Manager.ID) {
			oldVal := "none"
			if old.Manager != nil {
				oldVal = old.Manager.ID.String()
			}
			changes = append(changes, &FieldChange{ActionManagerChanged, oldVal, dto.ManagerID.String()})
		} else if dto.ManagerID == nil && old.Manager != nil {
			changes = append(changes, &FieldChange{ActionManagerChanged, old.Manager.ID.String(), "none"})
		}
	}

	if dto.HasField("assigneeId") {
		if dto.AssigneeID != nil && (old.Assignee == nil || *dto.AssigneeID != old.Assignee.ID) {
			oldVal := "none"
			if old.Assignee != nil {
				oldVal = old.Assignee.ID.String()
				changes = append(changes, &FieldChange{ActionAssignChanged, oldVal, dto.AssigneeID.String()})
			} else {
				changes = append(changes, &FieldChange{ActionAssigned, oldVal, dto.AssigneeID.String()})
			}
		} else if dto.AssigneeID == nil && old.Assignee != nil {
			changes = append(changes, &FieldChange{ActionAssignChanged, old.Assignee.ID.String(), "none"})
		}
	}
	if dto.HasField("ownerId") {
		if dto.OwnerID != nil && (old.Owner == nil || *dto.OwnerID != old.Owner.ID) {
			oldVal := "none"
			if old.Owner != nil {
				oldVal = old.Owner.ID.String()
			}
			changes = append(changes, &FieldChange{ActionOwnerChanged, oldVal, dto.OwnerID.String()})
		} else if dto.OwnerID == nil && old.Owner != nil {
			changes = append(changes, &FieldChange{ActionOwnerChanged, old.Owner.ID.String(), "none"})
		}
	}
	if dto.HasField("ownerMuted") && dto.OwnerMuted != old.OwnerMuted {
		changes = append(changes, &FieldChange{ActionOwnerMutedChanged, formatFieldValue(old.OwnerMuted), formatFieldValue(dto.OwnerMuted)})
	}
	if dto.HasField("equipmentId") {
		oldVal := "none"
		if old.Equipment != nil {
			oldVal = old.Equipment.ID.String()
		}
		newVal := "none"
		if dto.EquipmentID != nil {
			newVal = dto.EquipmentID.String()
		}
		if oldVal != newVal {
			changes = append(changes, &FieldChange{ActionEquipmentChanged, oldVal, newVal})
		}
	}

	return changes
}
