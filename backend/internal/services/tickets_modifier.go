package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// createModifierInput — вход разовой подстановки параметров подкатегории при
// создании заявки. Флаги прав считаются вызывающим (там же, где они уже
// понадобились для других полей), чтобы не делать одни и те же проверки дважды.
// IsExecutor — участие хотя бы в одной группе реалма, а не «назначен исполнителем
// этой заявки»: подкатегорию видит исполнитель независимо от того, кого в форме
// выбрали исполнителем новой заявки.
type createModifierInput struct {
	DTO        *models.TicketDTO
	Supervisor bool
	IsExecutor bool
	// Auto — подкатегория выбрана сервером по ключевым словам, а не клиентом.
	// Такой выбор не требует прав на ручной выбор; разовая подстановка
	// приоритета/срока при этом работает как обычно.
	Auto bool
}

// updateModifierInput — вход подготовки подкатегории при обновлении заявки.
type updateModifierInput struct {
	DTO     *models.TicketDTO
	Old     *models.Ticket
	ActorID uuid.UUID
}

// modifierPrepareResult — итог подготовки: Applied показывает, что параметры
// подкатегории только что разово применены, MayOverride — что актору разрешено
// переопределять срок/приоритет на заявке с подкатегорией.
type modifierPrepareResult struct {
	Changed     bool
	Applied     bool
	MayOverride bool
}

// modifierValuesInput — параметры разовой подстановки. Base — дата, от которой
// отсчитывается срок подкатегории: при создании это момент создания, при смене
// или добавлении подкатегории — created_at уже существующей заявки.
type modifierValuesInput struct {
	DTO         *models.TicketDTO
	Modifier    *models.CategoryModifier
	Category    *models.Category
	MayOverride bool
	Base        time.Time
	Calendar    WorkCalendar
}

func modifierID(t *models.Ticket) *uuid.UUID {
	if t == nil || t.CategoryModifier == nil {
		return nil
	}
	id := t.CategoryModifier.ID
	return &id
}

func sameModifierID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// findModifierByKeywords — общая точка подбора для создания заявки и для
// предпросмотра в форме. Ошибка справочника не блокирует операцию: автоподбор
// это удобство, а не обязательное поле, поэтому здесь только Warn-лог.
func (s *TicketService) findModifierByKeywords(ctx context.Context, realmID, categoryID uuid.UUID, title, description string) (*models.CategoryModifier, string) {
	if s.modifiers == nil || categoryID == uuid.Nil {
		return nil, ""
	}
	modifiers, err := s.modifiers.Get(ctx, &models.GetCategoryModifiersDTO{
		RealmID: realmID, CategoryID: &categoryID,
	})
	if err != nil {
		logger.Warn("failed to load category modifiers for keyword matching", logger.ErrAttr(err))
		return nil, ""
	}
	return matchModifierByKeywords(title, description, modifiers)
}

// autoSelectModifierByKeywords подбирает подкатегорию по словам заголовка и
// описания новой заявки. Это серверное значение по умолчанию: явный выбор
// клиента всегда важнее и не приравнивается к полю, присланному запросом
// (Provided у categoryModifierId не выставляется).
func (s *TicketService) autoSelectModifierByKeywords(ctx context.Context, dto *models.TicketDTO) bool {
	if dto.RealmID == nil || dto.CategoryID == uuid.Nil {
		return false
	}
	if dto.CategoryModifierID != nil && *dto.CategoryModifierID != uuid.Nil {
		return false
	}

	modifier, keyword := s.findModifierByKeywords(ctx, *dto.RealmID, dto.CategoryID, dto.Title, dto.Description)
	if modifier == nil {
		return false
	}
	id := modifier.ID
	dto.CategoryModifierID = &id
	logger.Info("category modifier selected by keywords",
		logger.StringAttr("realm_id", dto.RealmID.String()),
		logger.StringAttr("modifier_id", id.String()),
		logger.StringAttr("keyword", keyword),
	)
	return true
}

// PreviewModifierByKeywords отвечает форме создания тем же выбором, который
// сервер сделает при сохранении. Реалм приходит из заголовка (см. хендлер),
// категория — из тела; чужой реалм ничего не вернёт, т.к. справочник грузится
// только в его границах.
func (s *TicketService) PreviewModifierByKeywords(ctx context.Context, dto *models.ModifierPreviewDTO) (*models.CategoryModifierShort, error) {
	if dto.CategoryID == uuid.Nil {
		return nil, models.ErrInvalidInput
	}
	modifier, _ := s.findModifierByKeywords(ctx, dto.RealmID, dto.CategoryID, dto.Title, dto.Description)
	if modifier == nil {
		return nil, nil
	}
	return &models.CategoryModifierShort{ID: modifier.ID, Name: modifier.Name}, nil
}

// prepareCreateModifier проверяет выбор подкатегории и разово применяет её
// параметры. При создании подкатегорию выбирает исполнитель (участник хотя бы
// одной группы реалма) или начальник области. Право не зависит от того, кто
// назначен исполнителем новой заявки: исполнитель видит поле даже когда там
// выбран другой человек. Явные приоритет/срок сохраняются у тех же ролей —
// исполнителя и начальника области; остальным сервер подставляет параметры.
func (s *TicketService) prepareCreateModifier(ctx context.Context, in *createModifierInput) error {
	dto := in.DTO
	if dto.CategoryModifierID == nil || *dto.CategoryModifierID == uuid.Nil {
		return nil
	}
	if s.modifiers == nil || dto.RealmID == nil || dto.Actor == nil {
		return models.ErrInvalidInput
	}

	// Автоподбор — серверное значение по умолчанию, доступное всем создателям.
	// Ручной выбор по-прежнему только у исполнителя и начальника области.
	if !in.Auto && !in.Supervisor && !in.IsExecutor {
		return models.ErrPermissionDenied
	}

	modifier, err := s.modifiers.GetByID(ctx, &models.GetCategoryModifierByIDDTO{
		ID: *dto.CategoryModifierID, RealmID: *dto.RealmID,
	})
	if err != nil {
		return err
	}
	if modifier.CategoryID != dto.CategoryID || !modifier.IsActive {
		return models.ErrInvalidInput
	}

	category, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: dto.CategoryID, RealmID: *dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get category for modifier: %w", err)
	}

	// Исполнитель и начальник области оставляют свои явные срок/приоритет;
	// подкатегория подставляет значения только там, где они не заданы.
	// Отсчёт идёт от момента создания — для новой заявки это сейчас.
	return applyModifierValues(&modifierValuesInput{
		DTO: dto, Modifier: modifier, Category: category,
		MayOverride: in.Supervisor || in.IsExecutor,
		Base:        time.Now(),
		Calendar:    s.calendar,
	})
}

// prepareUpdateModifier подготавливает подкатегорию и её разовые параметры.
// Само поле меняется только в open и только исполнителем или начальником области;
// при смене категории подкатегория сбрасывается сервером, даже если клиент
// прислал другое значение.
func (s *TicketService) prepareUpdateModifier(ctx context.Context, in *updateModifierInput) (*modifierPrepareResult, error) {
	dto, old := in.DTO, in.Old
	result := &modifierPrepareResult{}
	if s.modifiers == nil || old.RealmID == nil {
		return result, nil
	}

	oldID := modifierID(old)
	newID := oldID
	if dto.HasField("categoryId") && old.Category != nil && dto.CategoryID != old.Category.ID && oldID != nil {
		newID = nil
		dto.CategoryModifierID = nil
		dto.MarkProvided("categoryModifierId")
	} else if dto.HasField("categoryModifierId") {
		newID = dto.CategoryModifierID
	}
	realm := ticketRealm(old)
	supervisor, err := s.access.IsRealmSupervisor(ctx, in.ActorID, realm)
	if err != nil {
		return result, fmt.Errorf("failed to check supervisor for modifier: %w", err)
	}
	isManager := false
	if old.Group != nil {
		isManager, err = s.isManagerOfGroup(ctx, old.Group, in.ActorID)
		if err != nil {
			return result, fmt.Errorf("failed to check manager for modifier: %w", err)
		}
	}
	executorCreator := old.Creator.ID == in.ActorID && old.Assignee != nil &&
		old.Assignee.ID == in.ActorID &&
		(dto.AssigneeID == nil || *dto.AssigneeID == in.ActorID)
	result.MayOverride = supervisor || isManager || executorCreator

	result.Changed = !sameModifierID(oldID, newID)
	if !result.Changed {
		return result, nil
	}
	if old.Status != models.StatusOpen {
		return result, models.ErrTicketFrozen
	}
	if newID != nil && *newID != uuid.Nil {
		if !supervisor && (old.Assignee == nil || old.Assignee.ID != in.ActorID) {
			return result, models.ErrPermissionDenied
		}
	}

	if newID == nil || *newID == uuid.Nil {
		return result, nil
	}

	modifier, err := s.modifiers.GetByID(ctx, &models.GetCategoryModifierByIDDTO{ID: *newID, RealmID: *old.RealmID})
	if err != nil {
		return result, err
	}
	if !modifier.IsActive {
		return result, models.ErrInvalidInput
	}
	categoryID := dto.CategoryID
	if !dto.HasField("categoryId") && old.Category != nil {
		categoryID = old.Category.ID
	}
	if modifier.CategoryID != categoryID {
		return result, models.ErrInvalidInput
	}

	category, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: categoryID, RealmID: *old.RealmID})
	if err != nil {
		return result, fmt.Errorf("failed to get category for modifier: %w", err)
	}
	// Дата отсчёта — creation date заявки, а не момент смены подкатегории.
	// Иначе поздний выбор подкатегории сдвигал бы срок вперёд и заявка
	// «омолаживалась»: N дней считались бы от сегодня, а не от создания.
	base := old.CreatedAt
	if base.IsZero() {
		base = time.Now()
	}
	if err := applyModifierValues(&modifierValuesInput{
		DTO: dto, Modifier: modifier, Category: category,
		MayOverride: result.MayOverride,
		Base:        base,
		Calendar:    s.calendar,
	}); err != nil {
		return result, err
	}
	result.Applied = true
	return result, nil
}

// applyModifierValues разово подставляет срок и приоритет. NULL у подкатегории
// означает наследование значения категории. Явные значения остаются только у
// тех, кому разрешено переопределение; иначе они затираются параметрами.
// Срок считается от in.Base — даты создания заявки (см. modifierValuesInput).
func applyModifierValues(in *modifierValuesInput) error {
	dto := in.DTO
	// Приоритет защищён тем же правилом, что и срок: начальник области,
	// менеджер группы и исполнитель-создатель могут оставить своё явное
	// значение, остальным сервер подставляет значение подкатегории (или
	// категории, если у подкатегории приоритет не задан).
	if !in.MayOverride || !dto.HasField("priority") {
		if in.Modifier.Priority != nil {
			dto.Priority = *in.Modifier.Priority
		} else if in.Category != nil {
			dto.Priority = in.Category.Priority
		}
		dto.MarkProvided("priority")
	}

	if !in.MayOverride || !dto.HasField("dueDate") {
		days := 0
		if in.Modifier.DefaultDueDays != nil {
			days = *in.Modifier.DefaultDueDays
		} else if in.Category != nil && in.Category.DefaultDueDays != nil {
			days = *in.Category.DefaultDueDays
		}
		if days > 0 {
			var dd time.Time
			if in.Calendar != nil {
				dd = in.Calendar.AddWorkingDays(in.Base, days)
			} else {
				dd = addWorkingDaysMonFri(in.Base, days, time.Local)
			}
			dto.DueDate = &dd
		} else {
			dto.DueDate = nil
		}
		dto.MarkProvided("dueDate")
	}
	return nil
}
