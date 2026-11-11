package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// EquipmentPeriodicTaskService — сервис управления периодическими задачами оборудования.
type EquipmentPeriodicTaskService struct {
	repo         postgres.Equipment
	tm           TransactionManager
	tickets      TicketCreator
	categories   Categories
	users        Users
	workingHours *workingHours
}

// NewEquipmentPeriodicTaskService создаёт сервис периодических задач.
func NewEquipmentPeriodicTaskService(repo postgres.Equipment, tm TransactionManager) *EquipmentPeriodicTaskService {
	return &EquipmentPeriodicTaskService{
		repo: repo,
		tm:   tm,
	}
}

// SetTicketCreator подключает создателя заявок (TicketService): периодическая
// задача — это создание заявки через обычный TicketService.Create, поэтому права,
// журнал и уведомления применяются штатно.
func (s *EquipmentPeriodicTaskService) SetTicketCreator(tc TicketCreator) {
	s.tickets = tc
}

// SetCategories подключает справочники категорий: из категории (группа, приоритет,
// автосрок) собираются реквизиты создаваемой заявки.
func (s *EquipmentPeriodicTaskService) SetCategories(c Categories) {
	s.categories = c
}

// SetUsers подключает пользователей: имя автора шаблона проставляется в Actor заявки.
func (s *EquipmentPeriodicTaskService) SetUsers(u Users) {
	s.users = u
}

// SetWorkingHours задаёт конфигурацию рабочего времени для планировщика.
func (s *EquipmentPeriodicTaskService) SetWorkingHours(wh *workingHours) {
	s.workingHours = wh
}

// PeriodicTaskStarter — подключается из EquipmentService после коммита выдачи:
// у типа с autoStartOnFirstAssign при первой выдаче единицы должна создаваться
// первая pending-задача (старт = дата выдачи + период).
type PeriodicTaskStarter interface {
	StartPeriodicTasksForAssignment(ctx context.Context, equipmentID, realmID uuid.UUID, assignedAt time.Time) error
}

var _ PeriodicTaskStarter = (*EquipmentPeriodicTaskService)(nil)

// StartPeriodicTasksForAssignment заводит первую pending-задачу для выданной
// единицы: включённые шаблоны типа с autoStartOnFirstAssign, у которых для пары
// «шаблон — единица» ещё нет экземпляров. Старт планируется от даты выдачи
// (assignedAt) + период; у единиц без выдачи (нет присвоения) старт не планируется.
func (s *EquipmentPeriodicTaskService) StartPeriodicTasksForAssignment(ctx context.Context, equipmentID, realmID uuid.UUID, assignedAt time.Time) error {
	if s.repo == nil {
		return nil
	}

	eq, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: equipmentID, RealmID: realmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment for periodic tasks: %w", err)
	}
	if eq.TypeID == nil || *eq.TypeID == uuid.Nil {
		// У комплекта нет типа — правил для него не существует.
		return nil
	}

	templates, err := s.repo.GetPeriodicTaskTemplates(ctx, realmID, eq.TypeID)
	if err != nil {
		return fmt.Errorf("failed to get periodic task templates: %w", err)
	}

	for _, template := range templates {
		if !template.Enabled || !template.AutoStartOnFirstAssign || template.CreatedBy == nil {
			continue
		}
		started, err := s.repo.HasPeriodicTaskInstances(ctx, template.ID, equipmentID, realmID)
		if err != nil {
			return fmt.Errorf("failed to check periodic task instances: %w", err)
		}
		if started {
			continue
		}

		assignment, err := s.repo.GetLastAssignmentForEquipment(ctx, equipmentID)
		if err != nil {
			return fmt.Errorf("failed to get last assignment: %w", err)
		}
		startAt := assignedAt
		if assignment != nil && !assignment.AssignedAt.IsZero() {
			startAt = assignment.AssignedAt
		}
		lastAssignmentID := (*uuid.UUID)(nil)
		if assignment != nil {
			lastAssignmentID = &assignment.ID
		}

		instance := &models.EquipmentPeriodicTaskInstance{
			ID:               uuid.New(),
			RealmID:          template.RealmID,
			TemplateID:       template.ID,
			EquipmentID:      eq.ID,
			Status:           models.TaskInstancePending,
			ScheduledAt:      startAt.AddDate(template.PeriodYears, template.PeriodMonths, template.PeriodDays),
			LastAssignmentID: lastAssignmentID,
			CreatedBy:        template.CreatedBy,
		}
		if err := s.repo.CreatePeriodicTaskInstance(ctx, nil, instance); err != nil {
			return fmt.Errorf("failed to create initial periodic task instance: %w", err)
		}
	}
	return nil
}

// GetPeriodicTaskTemplates возвращает шаблоны периодических задач реалма.
// Если equipmentTypeID != nil, фильтрует по типу.
func (s *EquipmentPeriodicTaskService) GetPeriodicTaskTemplates(ctx context.Context, realmID uuid.UUID, equipmentTypeID *uuid.UUID) ([]*models.EquipmentTypePeriodicTask, error) {
	return s.repo.GetPeriodicTaskTemplates(ctx, realmID, equipmentTypeID)
}

// GetPeriodicTaskTemplateByID возвращает шаблон по ID.
func (s *EquipmentPeriodicTaskService) GetPeriodicTaskTemplateByID(ctx context.Context, id uuid.UUID, realmID uuid.UUID) (*models.EquipmentTypePeriodicTask, error) {
	return s.repo.GetPeriodicTaskTemplateByID(ctx, id, realmID)
}

// CreatePeriodicTaskTemplate создаёт шаблон периодической задачи.
func (s *EquipmentPeriodicTaskService) CreatePeriodicTaskTemplate(ctx context.Context, dto *models.EquipmentTypePeriodicTaskDTO) (*models.EquipmentTypePeriodicTask, error) {
	if dto.Title == "" {
		return nil, models.ErrNotValid
	}
	if dto.CategoryID == uuid.Nil {
		return nil, models.ErrNotValid
	}
	if dto.RealmID == uuid.Nil {
		return nil, models.ErrNotValid
	}
	if dto.EquipmentTypeID == uuid.Nil {
		return nil, models.ErrNotValid
	}
	if !hasPeriod(dto.PeriodYears, dto.PeriodMonths, dto.PeriodDays) {
		return nil, models.ErrNotValid
	}

	var template *models.EquipmentTypePeriodicTask
	var createErr error
	createErr = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		var txErr error
		template, txErr = s.repo.CreatePeriodicTaskTemplate(ctx, tx, dto)
		if txErr != nil {
			return txErr
		}

		// Если autoStartOnFirstAssign = true, создаём pending-задачи для всех единиц
		// типа, у которых ещё нет цепочки. Автор шаблона — создатель заявки.
		if dto.AutoStartOnFirstAssign && dto.CreatedBy != nil {
			if createErr = s.createInitialInstancesForType(ctx, tx, template, dto.CreatedBy); createErr != nil {
				return fmt.Errorf("failed to create initial instances: %w", createErr)
			}
		}

		return nil
	})
	if createErr != nil {
		return nil, createErr
	}
	return template, nil
}

// createInitialInstancesForType создаёт pending-задачи для всех единиц типа с
// последней выдачей (старт = дата выдачи + период). Единицы без выдачи пропускаются:
// первый старт для них вручную.
func (s *EquipmentPeriodicTaskService) createInitialInstancesForType(ctx context.Context, tx postgres.Tx, template *models.EquipmentTypePeriodicTask, initiatorID *uuid.UUID) error {
	equipment, _, err := s.repo.Get(ctx, &models.EquipmentFilter{
		RealmID: template.RealmID,
		TypeID:  &template.EquipmentTypeID,
		Limit:   1000,
	})
	if err != nil {
		return fmt.Errorf("failed to get equipment: %w", err)
	}

	for _, eq := range equipment {
		assignment, err := s.repo.GetLastAssignmentForEquipment(ctx, eq.ID)
		if err != nil {
			return fmt.Errorf("failed to get assignment for equipment %s: %w", eq.ID, err)
		}
		if assignment == nil || assignment.AssignedAt.IsZero() {
			// Нет выдачи — первый старт не планируем.
			continue
		}

		instance := &models.EquipmentPeriodicTaskInstance{
			ID:               uuid.New(),
			RealmID:          template.RealmID,
			TemplateID:       template.ID,
			EquipmentID:      eq.ID,
			Status:           models.TaskInstancePending,
			ScheduledAt:      assignment.AssignedAt.AddDate(template.PeriodYears, template.PeriodMonths, template.PeriodDays),
			LastAssignmentID: &assignment.ID,
			CreatedBy:        initiatorID,
		}
		if err := s.repo.CreatePeriodicTaskInstance(ctx, tx, instance); err != nil {
			return fmt.Errorf("failed to create instance: %w", err)
		}
	}

	return nil
}

// UpdatePeriodicTaskTemplate обновляет шаблон периодической задачи. Автор
// (created_by) сохраняется: при пустом CreatedBy (обновление старого шаблона)
// COALESCE оставляет прежнего.
func (s *EquipmentPeriodicTaskService) UpdatePeriodicTaskTemplate(ctx context.Context, dto *models.EquipmentTypePeriodicTaskDTO) (*models.EquipmentTypePeriodicTask, error) {
	if dto.Title == "" {
		return nil, models.ErrNotValid
	}
	if dto.CategoryID == uuid.Nil {
		return nil, models.ErrNotValid
	}
	if dto.RealmID == uuid.Nil {
		return nil, models.ErrNotValid
	}
	if dto.EquipmentTypeID == uuid.Nil {
		return nil, models.ErrNotValid
	}
	if !hasPeriod(dto.PeriodYears, dto.PeriodMonths, dto.PeriodDays) {
		return nil, models.ErrNotValid
	}

	var template *models.EquipmentTypePeriodicTask
	var updateErr error
	updateErr = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		var txErr error
		template, txErr = s.repo.UpdatePeriodicTaskTemplate(ctx, tx, dto)
		return txErr
	})
	if updateErr != nil {
		return nil, updateErr
	}
	return template, nil
}

// DeletePeriodicTaskTemplate удаляет шаблон периодической задачи.
func (s *EquipmentPeriodicTaskService) DeletePeriodicTaskTemplate(ctx context.Context, dto *models.DelEquipmentTypePeriodicTaskDTO) error {
	if dto.ID == uuid.Nil {
		return models.ErrNotValid
	}
	return s.repo.DeletePeriodicTaskTemplate(ctx, dto)
}

// GetPeriodicTaskInstances возвращает экземпляры периодических задач.
func (s *EquipmentPeriodicTaskService) GetPeriodicTaskInstances(ctx context.Context, filter *models.GetEquipmentPeriodicTaskInstancesFilter) ([]*models.EquipmentPeriodicTaskInstance, int, error) {
	return s.repo.GetPeriodicTaskInstances(ctx, filter)
}

// ProcessPendingTasks обрабатывает pending-задачи: создаёт заявки и планирует
// следующие. Ошибки одного экземпляра не останавливают обработку остальных.
func (s *EquipmentPeriodicTaskService) ProcessPendingTasks(ctx context.Context) error {
	now := time.Now().UTC()

	// Проверяем рабочее время.
	if s.workingHours != nil && !s.workingHours.allows(now) {
		return nil
	}

	pending, err := s.repo.GetPendingPeriodicTasks(ctx, now, 100)
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}

	for _, instance := range pending {
		if err := s.processSingleTask(ctx, instance, now); err != nil {
			logger.Error("failed to process equipment periodic task",
				logger.StringAttr("instance_id", instance.ID.String()),
				logger.StringAttr("equipment_id", instance.EquipmentID.String()),
				logger.ErrAttr(err),
			)
		}
	}

	return nil
}

// processSingleTask обрабатывает одну pending-задачу: claim → создание заявки →
// планирование следующего запуска. При сбое заявки claim откатывается, задача
// повторится на следующем тике.
func (s *EquipmentPeriodicTaskService) processSingleTask(ctx context.Context, instance *models.EquipmentPeriodicTaskInstance, now time.Time) error {
	if instance.Status != models.TaskInstancePending {
		return nil
	}

	template, err := s.repo.GetPeriodicTaskTemplateByID(ctx, instance.TemplateID, instance.RealmID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}
	if !template.Enabled {
		// Шаблон выключен: тик не должен стартить задачи.
		return nil
	}

	eq, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: instance.EquipmentID, RealmID: instance.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment: %w", err)
	}

	nextAt := instance.ScheduledAt.AddDate(template.PeriodYears, template.PeriodMonths, template.PeriodDays)

	// Claim-first: атомарный перевод pending → created. Дедупликация конкуррирующих
	// тиков — через RowsAffected claim'а.
	claimed, err := s.repo.ClaimPeriodicTaskInstance(ctx, nil, instance.ID, instance.RealmID)
	if err != nil {
		return fmt.Errorf("failed to claim periodic task instance: %w", err)
	}
	if !claimed {
		// Уже обработан (конкурентный тик): пропускаем.
		return nil
	}

	ticketDTO, err := s.buildTicket(ctx, template, eq)
	if err != nil {
		if revertErr := s.repo.RevertPeriodicTaskClaim(ctx, nil, instance.ID, instance.RealmID, instance.ScheduledAt, instance.LastAssignmentID); revertErr != nil {
			return fmt.Errorf("%w (revert failed: %v)", err, revertErr)
		}
		return err
	}

	if err := s.tickets.Create(ctx, ticketDTO); err != nil {
		if revertErr := s.repo.RevertPeriodicTaskClaim(ctx, nil, instance.ID, instance.RealmID, instance.ScheduledAt, instance.LastAssignmentID); revertErr != nil {
			return fmt.Errorf("failed to create ticket: %w (revert failed: %v)", err, revertErr)
		}
		return fmt.Errorf("failed to create ticket: %w", err)
	}
	if ticketDTO.ID == nil {
		if revertErr := s.repo.RevertPeriodicTaskClaim(ctx, nil, instance.ID, instance.RealmID, instance.ScheduledAt, instance.LastAssignmentID); revertErr != nil {
			return fmt.Errorf("ticket ID is nil after creation (revert failed: %v)", revertErr)
		}
		return fmt.Errorf("ticket ID is nil after creation")
	}

	// Заявка создана: привязываем её к экземпляру и плановируем следующий запуск
	// (якорь — расписание, а не now: интервалы месяцев/лет не должны дрейфовать от
	// момента обработки).
	if err := s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.SetPeriodicTaskInstanceTicket(ctx, tx, instance, *ticketDTO.ID, nextAt); err != nil {
			return err
		}
		nextInstance := &models.EquipmentPeriodicTaskInstance{
			ID:               uuid.New(),
			RealmID:          instance.RealmID,
			TemplateID:       instance.TemplateID,
			EquipmentID:      instance.EquipmentID,
			Status:           models.TaskInstancePending,
			ScheduledAt:      nextAt,
			LastAssignmentID: instance.LastAssignmentID,
			CreatedBy:        template.CreatedBy,
		}
		return s.repo.CreatePeriodicTaskInstance(ctx, tx, nextInstance)
	}); err != nil {
		return fmt.Errorf("failed to schedule next periodic task: %w", err)
	}

	return nil
}

// buildTicket собирает TicketDTO для заявки, порождаемой шаблоном. Создателем
// (и, соответственно, заказчиком/исполнителем по правилам TicketService) становится
// автор шаблона: recurring-задача живёт в его правах. Группа — из категории.
func (s *EquipmentPeriodicTaskService) buildTicket(ctx context.Context, template *models.EquipmentTypePeriodicTask, eq *models.Equipment) (*models.TicketDTO, error) {
	authorID := uuid.Nil
	if template.CreatedBy != nil {
		authorID = *template.CreatedBy
	}
	if authorID == uuid.Nil {
		return nil, fmt.Errorf("periodic task template %s has no author", template.ID)
	}

	category, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: template.CategoryID, RealmID: template.RealmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	if category.GroupID == uuid.Nil {
		return nil, fmt.Errorf("periodic task category %s has no group", template.CategoryID)
	}

	siteID := uuid.Nil
	if template.SiteID != nil {
		siteID = *template.SiteID
	}

	authorName := s.authorName(ctx, authorID)
	ticketDTO := &models.TicketDTO{
		Actor: &models.Actor{
			ID:   authorID,
			Name: authorName,
		},
		RealmID:     &template.RealmID,
		Title:       template.Title,
		Description: fmt.Sprintf("%s\n\nОборудование: %s (инв. № %s)", template.Description, eq.Name, inventoryLabel(eq)),
		Status:      models.StatusOpen,
		Priority:    models.Priority(template.Priority),
		CategoryID:  template.CategoryID,
		GroupID:     &category.GroupID,
		// Заказчик и создатель — автор шаблона: заявка «для него», и права на создание
		// (которые у автора шаблона должны быть) проверяет TicketService.
		OwnerID:     &authorID,
		CreatorID:   authorID,
		SiteID:      siteID,
		EquipmentID: &eq.ID,
	}
	return ticketDTO, nil
}

// authorName собирает человекочитаемое имя автора по ID (best-effort: без пользователя
// остаётся пустая строка, права всё равно резолвятся по ID).
func (s *EquipmentPeriodicTaskService) authorName(ctx context.Context, userID uuid.UUID) string {
	if s.users == nil {
		return ""
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil {
		return ""
	}
	name := strings.TrimSpace(user.LastName + " " + user.FirstName)
	if user.Username != "" {
		name = fmt.Sprintf("%s (%s)", name, user.Username)
	}
	return name
}

// ManualRunTask запускает периодическую задачу вручную: всегда создаёт заявку
// сейчас. Семантика цепочки:
//   - shift=true — существующий pending перепланируется на now + период, заявка
//     создаётся сейчас и цепочка продолжается с нового расписания;
//   - shift=false — pending остаётся на исходном расписании (при отсутствии pending
//     создаётся новый старт на now + период — цепочка обязана стартовать);
//   - pending нет вообще — создаём старт на now + период и заявку сейчас.
func (s *EquipmentPeriodicTaskService) ManualRunTask(ctx context.Context, dto *models.ManualPeriodicTaskRunDTO, actor *models.Actor) error {
	if actor == nil {
		return models.ErrPermissionDenied
	}

	template, err := s.repo.GetPeriodicTaskTemplateByID(ctx, dto.TemplateID, dto.RealmID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}
	if !template.Enabled {
		return models.ErrNotValid
	}

	eq, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.EquipmentID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment: %w", err)
	}

	now := time.Now().UTC()

	pending, err := s.repo.GetPendingPeriodicTaskInstance(ctx, dto.TemplateID, dto.EquipmentID, dto.RealmID)
	if err != nil {
		return fmt.Errorf("failed to get pending periodic task: %w", err)
	}

	shiftNext := dto.ShiftNext
	pendingFromDB := pending != nil
	if pending == nil {
		// pending нет — первая задача цепочки: старт на now, создаём экземпляр
		// (цепочка обязана стартовать).
		assignment, err := s.repo.GetLastAssignmentForEquipment(ctx, eq.ID)
		if err != nil {
			return fmt.Errorf("failed to get last assignment: %w", err)
		}
		lastAssignmentID := (*uuid.UUID)(nil)
		if assignment != nil {
			lastAssignmentID = &assignment.ID
		}
		pending = &models.EquipmentPeriodicTaskInstance{
			ID:               uuid.New(),
			RealmID:          template.RealmID,
			TemplateID:       template.ID,
			EquipmentID:      eq.ID,
			Status:           models.TaskInstancePending,
			ScheduledAt:      now,
			LastAssignmentID: lastAssignmentID,
			CreatedBy:        &actor.ID,
		}
		if err := s.repo.CreatePeriodicTaskInstance(ctx, nil, pending); err != nil {
			return fmt.Errorf("failed to create periodic task instance: %w", err)
		}
	}

	// Запуск всегда создаёт заявку сейчас: актор — инициатор ручного запуска.
	ticketDTO, err := s.buildTicket(ctx, template, eq)
	if err != nil {
		return err
	}
	ticketDTO.Actor = actor
	ticketDTO.CreatorID = actor.ID
	ticketDTO.OwnerID = &actor.ID

	if err := s.tickets.Create(ctx, ticketDTO); err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}
	if ticketDTO.ID == nil {
		return fmt.Errorf("ticket ID is nil after creation")
	}
	ticketID := *ticketDTO.ID

	// Сдвиг: при shift=true расписание pending переписывается на now + период
	// (следующий автоматический запуск уходит вперёд); при shift=false pending
	// остаётся на исходном расписании.
	if shiftNext && pendingFromDB {
		if err := s.repo.ReschedulePendingInstance(ctx, nil, pending, now.AddDate(template.PeriodYears, template.PeriodMonths, template.PeriodDays)); err != nil {
			return fmt.Errorf("failed to reschedule pending periodic task: %w", err)
		}
		pending.ScheduledAt = now.AddDate(template.PeriodYears, template.PeriodMonths, template.PeriodDays)
	}

	// Заявка создана — привязываем экземпляр и планируем следующий старт:
	// следующий запуск = расписание текущего + период (якорь — расписание, не now).
	created := *pending
	created.Status = models.TaskInstanceCreated
	created.TicketID = &ticketID
	created.CreatedAt = &now
	nextAt := created.ScheduledAt.AddDate(template.PeriodYears, template.PeriodMonths, template.PeriodDays)
	created.NextScheduledAt = &nextAt
	if err := s.repo.UpdatePeriodicTaskInstance(ctx, nil, &created); err != nil {
		return fmt.Errorf("failed to update periodic task instance: %w", err)
	}

	nextInstance := &models.EquipmentPeriodicTaskInstance{
		ID:               uuid.New(),
		RealmID:          template.RealmID,
		TemplateID:       template.ID,
		EquipmentID:      eq.ID,
		Status:           models.TaskInstancePending,
		ScheduledAt:      nextAt,
		LastAssignmentID: created.LastAssignmentID,
		CreatedBy:        created.CreatedBy,
	}
	return s.repo.CreatePeriodicTaskInstance(ctx, nil, nextInstance)
}

// hasPeriod сообщает, что период задан хотя бы одним ненулевым компонентом
// (годы/месяцы/дни): все нули — невалидный шаблон.
func hasPeriod(years, months, days int) bool {
	return years > 0 || months > 0 || days > 0
}

// inventoryLabel собирает человекочитаемый номер единицы для описания заявки.
func inventoryLabel(eq *models.Equipment) string {
	if eq.InventoryNumber == nil || *eq.InventoryNumber == "" {
		return "не указан"
	}
	return *eq.InventoryNumber
}
