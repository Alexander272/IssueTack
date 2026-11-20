package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
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

	if !in.Supervisor && !in.IsExecutor {
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
		Base:        time.Now().UTC(),
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
		base = time.Now().UTC()
	}
	if err := applyModifierValues(&modifierValuesInput{
		DTO: dto, Modifier: modifier, Category: category,
		MayOverride: result.MayOverride,
		Base:        base,
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
			dd := in.Base.UTC().AddDate(0, 0, days)
			dto.DueDate = &dd
		} else {
			dto.DueDate = nil
		}
		dto.MarkProvided("dueDate")
	}
	return nil
}
