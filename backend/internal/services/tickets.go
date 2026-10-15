package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// TicketService — сервис тикетов: реализует бизнес-правила доступа,
// переходов по статусам, автозакрытия и журналирование действий.
type TicketService struct {
	repo          repository.Tickets
	tx            TransactionManager
	logs          ActivityLog
	subtasks      Subtasks
	attachments   Attachments
	notifications Notifications
	groups        Groups
	categories    Categories
	access        TicketAccessChecker
}

// NewTicketService создаёт новый сервис тикетов на основе переданных зависимостей.
func NewTicketService(deps *TicketDeps) *TicketService {
	return &TicketService{
		repo:          deps.Repo,
		tx:            deps.TxManager,
		logs:          deps.Logs,
		subtasks:      deps.Subtasks,
		attachments:   deps.Attachments,
		notifications: deps.Notifications,
		groups:        deps.Groups,
		categories:    deps.Categories,
		access:        deps.Access,
	}
}

// TicketDeps — зависимости, необходимые для создания TicketService.
type TicketDeps struct {
	Repo          repository.Tickets
	TxManager     TransactionManager
	Logs          ActivityLog
	Subtasks      Subtasks
	Attachments   Attachments
	Notifications Notifications
	Groups        Groups
	Categories    Categories
	Access        TicketAccessChecker
}

// Tickets — интерфейс сервиса тикетов: описывает чтение, создание,
// обновление и удаление тикетов, а также автоматическое закрытие.
type Tickets interface {
	// Get возвращает список тикетов с учётом прав актора и общее количество.
	Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error)
	// GetByID возвращает тикет по идентификатору вместе с подзадачами, вложениями и флагами доступа.
	GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error)
	// GetSummary возвращает «сырой» тикет по идентификатору — без подзадач, вложений,
	// флагов доступа и без проверки прав. Используется внутренними сервисами, которым
	// нужен сам тикет (статус, исполнитель, группа и т.п.), а не полный портрет с правами.
	GetSummary(ctx context.Context, id uuid.UUID) (*models.Ticket, error)
	// GetRealmIDByTicketID возвращает реалм заявки без проверки прав и без загрузки
	// самой заявки. Нужен там, где канал ещё неизвестен: реалм определяется по тикету,
	// чтобы затем выполнить полную проверку прав в GetByID.
	GetRealmIDByTicketID(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	// Create создаёт новый тикет.
	Create(ctx context.Context, dto *models.TicketDTO) error
	// Update изменяет поля тикета с соблюдением правил доступа и переходов по статусам.
	Update(ctx context.Context, dto *models.TicketDTO) error
	// Take назначает пользователя исполнителем заявки и переводит её в статус
	// in_progress («взять в работу»). Доступно, если заявка активна и «забираема»:
	// исполнителя нет, либо исполнитель — другой пользователь и статус открыт.
	Take(ctx context.Context, dto *models.TakeTicketDTO) error
	// Transfer передаёт заявку от текущего исполнителя другому участнику той же
	// группы. Статус не меняется.
	Transfer(ctx context.Context, dto *models.TransferTicketDTO) error
	// Delete удаляет тикет.
	Delete(ctx context.Context, dto *models.DeleteTicketDTO) error
	// UploadAttachment сохраняет вложение и рассылает уведомление о новом вложении
	// подписанным и исполнителю (для standalone-вложений без комментария; вложения,
	// привязанные к комментарию, уведомлением не дублируют TicketCommented).
	UploadAttachment(ctx context.Context, tx postgres.Tx, dto *models.UploadAttachmentDTO) (*models.Attachment, error)
	// AutoCloseResolved закрывает resolved-тикеты по истечении задержки и возвращает их количество.
	AutoCloseResolved(ctx context.Context, delay time.Duration) (int64, error)
}

// Get возвращает список тикетов и общее количество с учётом прав актора:
// при отсутствии повышенных прав список ограничивается его группами
// и тикетами, где он является создателем или исполнителем.
func (s *TicketService) Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error) {
	realmStr := ""
	if req.RealmID != nil {
		realmStr = req.RealmID.String()
	}

	// Для страницы «Избранное» отдаём только избранные пользователя без ограничений по
	// группам/режиму: пользователь уже имеет read-доступ к своим избранным заявкам.
	if req.FavoritesByUser != nil {
		data, total, err := s.repo.Get(ctx, req)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get tickets. error: %w", err)
		}
		if err := s.attachSubtasks(ctx, data); err != nil {
			return nil, 0, fmt.Errorf("failed to attach subtasks. error: %w", err)
		}
		return data, total, nil
	}

	canViewAll, err := s.access.IsRealmSupervisor(ctx, req.Actor.ID, realmStr)
	if err != nil {
		return nil, 0, fmt.Errorf("realm supervisor check failed: %w", err)
	}

	// «Мои задачи» (mode=assigned): лично назначенные ИЛИ задачи групп пользователя.
	// Работает и для рядовых пользователей, и для супервайзеров: супервайзер видит
	// в «Задачах» свои назначения и задачи групп, где состоит (а не все заявки реалма).
	// Здесь же resolved уходит в архив: список исполнителя — рабочий, и решённые
	// заявки (до подтверждения/авто-закрытия могут висеть неделями) его не засоряют.
	if req.Mode != nil && *req.Mode == "assigned" {
		req.ArchiveWithResolved = true
		member, err := s.groups.GetMemberGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get member groups: %w", err)
		}
		groupIDs := member
		if !canViewAll {
			managed, err := s.groups.GetManagedGroups(ctx, req.Actor.ID, nil)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get managed groups: %w", err)
			}
			groupIDs = unionGroupIDs(managed, member)
		}
		req.MyWork = &models.MyWorkFilter{UserID: req.Actor.ID, GroupIDs: groupIDs}
	} else if !canViewAll {
		managed, err := s.groups.GetManagedGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get managed groups: %w", err)
		}
		member, err := s.groups.GetMemberGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get member groups: %w", err)
		}

		// «Созданные вами заявки» (mode=created): только заявки, где пользователь автор,
		// без ограничения группой — иначе теряются внегрупповые заявки автора.
		if req.Mode != nil && *req.Mode == "created" {
			req.CreatorID = &req.Actor.ID
		} else if req.Mode != nil && *req.Mode == "created_or_owned" {
			// «Автор ИЛИ заказчик» (mode=created_or_owned, Mattermost-плагин):
			// тикеты, где пользователь автор ИЛИ заказчик (owner), без ограничения группой.
			req.CreatedOrOwnedBy = &req.Actor.ID
		} else {
			// Обзор без mode: все заявки групп пользователя + внегрупповые, назначенные ему.
			if groups := unionGroupIDs(managed, member); len(groups) > 0 {
				req.GroupIDs = groups
			}
			req.IncludeUngroupedAssignedTo = &req.Actor.ID
		}
	}

	data, total, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tickets. error: %w", err)
	}
	if err := s.attachSubtasks(ctx, data); err != nil {
		return nil, 0, fmt.Errorf("failed to attach subtasks: %w", err)
	}
	return data, total, nil
}

// unionGroupIDs объединяет два списка групп без дубликатов (порядок: managed, затем member).
func unionGroupIDs(managed, member []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(managed)+len(member))
	out := make([]uuid.UUID, 0, len(managed)+len(member))
	for _, gid := range managed {
		if _, ok := seen[gid]; !ok {
			seen[gid] = struct{}{}
			out = append(out, gid)
		}
	}
	for _, gid := range member {
		if _, ok := seen[gid]; !ok {
			seen[gid] = struct{}{}
			out = append(out, gid)
		}
	}
	return out
}

// attachSubtasks загружает подзадачи для всех заявок листинга одним запросом
// и раскладывает их по тикетам. Фронт листинга (TaskRow/TaskCard) рисует
// прогресс-бар и счётчик подзадач из task.subtasks. В детали (GetByID) подзадачи
// приходят отдельно через s.subtasks.GetByTicketID — там этот вызов не нужен.
func (s *TicketService) attachSubtasks(ctx context.Context, data []*models.Ticket) error {
	if len(data) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(data))
	for i, t := range data {
		ids[i] = t.ID
	}
	byTicket, err := s.subtasks.GetByTicketIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to get subtasks for tickets: %w", err)
	}
	for _, t := range data {
		t.Subtasks = byTicket[t.ID]
	}
	return nil
}

// GetSummary возвращает «сырой» тикет по идентификатору для внутренних сервисов
// (без подзадач, вложений, флагов доступа и проверки прав). Используется там, где
// нужен сам агрегат, а не полный портрет с правами: проверка статуса/исполнителя,
// сбор сведений для уведомлений и т.п.
func (s *TicketService) GetSummary(ctx context.Context, id uuid.UUID) (*models.Ticket, error) {
	data, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: id})
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket summary: %w", err)
	}
	return data, nil
}

// GetRealmIDByTicketID возвращает реалм заявки. Проверки прав здесь намеренно нет:
// вызывающий код использует результат только для того, чтобы построить
// GetTicketByIdDTO с корректным реалмом, а доступ проверяется уже в GetByID.
func (s *TicketService) GetRealmIDByTicketID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	realmID, err := s.repo.GetRealmIDByTicketID(ctx, id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get ticket realm: %w", err)
	}
	return realmID, nil
}

// GetByID возвращает тикет по идентификатору с проверкой права чтения,
// дополняя его подзадачами, вложениями и флагами доступа.
//
// Тикет грузится до проверки доступа, а не после: атрибутная модель (создатель,
// исполнитель, членство в группе) требует самого тикета, поэтому загрузка
// предшествует гейту. Так заодно уходит двойное чтение — раньше CheckAccess
// внутри себя грузил тикет, и следом его снова читал этот же метод.
func (s *TicketService) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket by id. error: %w", err)
	}

	if err := s.access.CheckAccessOnTicket(ctx, data, req.Actor.ID, string(access.Read)); err != nil {
		return nil, err
	}

	subtasks, err := s.subtasks.GetByTicketID(ctx, data.ID, req.Actor.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	data.Subtasks = subtasks

	attachments, err := s.attachments.GetByEntity(ctx, &models.EntityAccessDTO{
		EntityType: string(access.ResourceTicket),
		EntityID:   data.ID,
		ActorID:    req.Actor.ID,
		Realm:      req.RealmID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	data.Attachments = attachments

	flags, err := s.GetAccessFlags(ctx, data, req.Actor.ID)
	if err == nil {
		data.Access = flags
	}

	return data, nil
}

// notifyBestEffort отправляет уведомление после фикса тикета. Ошибки
// уведомления не должны превращать уже совершённую операцию в 500 для клиента,
// но обязательно сигнализируются разработчику (error_bot) и в лог.
func (s *TicketService) notifyBestEffort(action string, entityID uuid.UUID, entity string, notifyErr error) {
	if notifyErr == nil {
		return
	}
	errMsg := fmt.Sprintf("failed to send notification (%s): %v", action, notifyErr)
	logger.Error(errMsg,
		logger.StringAttr("entity_id", entityID.String()),
		logger.StringAttr("entity", entity),
	)
	error_bot.Send(nil, errMsg, map[string]string{
		action:      entity,
		"entity_id": entityID.String(),
	})
}
