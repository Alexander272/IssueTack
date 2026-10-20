package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// Assign выдаёт единицу (или комплект) держателю. Выдача пропагируется на весь
// состав. Если кто-то из состава уже выдан другому держателю — конфликт (обход
// только через force). Место выдача не меняет: пустые «Площадка»/«Локация» в теле
// означают «единица не переезжает», а явно указанное место важнее.
func (s *EquipmentService) Assign(ctx context.Context, dto *models.AssignEquipmentDTO) error {
	root, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if root.Status == models.EquipmentStatusWrittenOff {
		return models.ErrEquipmentNotAssignable
	}

	// Дата выдачи: из формы приходит точный момент (datetime-local), по
	// умолчанию — сейчас. На будущее задать нельзя: проверка до транзакции,
	// чтобы не занимать её при отказе. Прошлое разрешено — акт оформляют
	// позже факта.
	assignedAt := time.Now()
	if dto.AssignedAt != nil {
		if dto.AssignedAt.After(assignedAt) {
			return models.ErrEquipmentAssignDateInFuture
		}
		assignedAt = *dto.AssignedAt
	}

	// Место — реквизит единицы («где она лежит»), а не часть выдачи: пустые
	// «Площадка»/«Локация» в диалоге означают «единица не переезжает». Раньше они
	// затирали место (SetHolderAndStatus записывал nil), и вместе с местом пропадала
	// подпись площадки в снимке события и в акте. Явно указанное место по-прежнему
	// важнее: единицу могут выдать сразу на другой объект.
	if !hasHolder(dto) && placeAbsent(dto.SiteID, dto.Location) {
		return models.ErrEquipmentNoHoldersOrLocation
	}
	if placeAbsent(dto.SiteID, dto.Location) {
		dto.SiteID, dto.Location = root.SiteID, root.Location
	}

	affected, err := s.collectSubtree(ctx, dto.RealmID, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to collect equipment subtree. error: %w", err)
	}

	// Подписи для снимка события резолвятся до транзакции: ссылки наши же, транзакция
	// их не меняет, а внутри цикла они одинаковы для всего состава.
	holderFullName, siteName, err := s.resolveAssignmentLabels(ctx, dto.UserID, dto.SiteID)
	if err != nil {
		return err
	}

	for _, eq := range affected {
		open, err := s.repo.GetOpenAssignment(ctx, eq.ID)
		if err != nil && !errors.Is(err, models.ErrNotFound) {
			return fmt.Errorf("failed to get open assignment. error: %w", err)
		}
		if conflictsWith(open, dto) && !dto.Force {
			return models.ErrEquipmentAssigned
		}
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		opID := uuid.New()
		// Реальное время переоформления: им закрывается прежняя открытая выдача.
		// Новая запись получает assignedAt (момент из формы). При переоформлении
		// задним числом интервалы могут накладываться — осознанный компромисс:
		// закрывать чужую выдачу в прошлом хуже.
		now := time.Now()
		actorID := actorID(dto.Actor)

		for _, eq := range affected {
			open, err := s.repo.GetOpenAssignment(ctx, eq.ID)
			if err != nil && !errors.Is(err, models.ErrNotFound) {
				return fmt.Errorf("failed to get open assignment. error: %w", err)
			}
			if err == nil {
				if err := s.repo.CloseAssignment(ctx, tx, open.ID, now, ""); err != nil {
					return fmt.Errorf("failed to close assignment. error: %w", err)
				}
			}

			if err := s.repo.CreateAssignment(ctx, tx, &models.EquipmentAssignment{
				EquipmentID: eq.ID,
				UserID:      dto.UserID,
				HolderName:  dto.HolderName,
				SiteID:      dto.SiteID,
				Location:    dto.Location,
				AssignedAt:  assignedAt,
				AssignedBy:  actorID,
				Comment:     dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create assignment. error: %w", err)
			}

			if err := s.repo.SetHolderAndStatus(ctx, tx, eq.ID, dto.UserID, dto.HolderName, dto.SiteID, dto.Location, models.EquipmentStatusInUse); err != nil {
				return fmt.Errorf("failed to set holder. error: %w", err)
			}
			if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
				EquipmentID: eq.ID, EventType: models.EquipmentEventAssigned, OperationID: opID,
				NewValue: assignmentJSON(dto, holderFullName, siteName), UserID: actorID, Comment: dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create event. error: %w", err)
			}
		}

		if dto.SkipAct {
			return nil
		}
		return s.createActTx(ctx, tx, &models.EquipmentActDTO{
			RealmID:         dto.RealmID,
			KitEquipmentID:  &dto.ID,
			RecipientUserID: dto.UserID,
			RecipientName:   dto.HolderName,
			SiteID:          dto.SiteID,
			Location:        dto.Location,
			Comment:         dto.Comment,
			IssuedAt:        &assignedAt,
			Actor:           dto.Actor,
		}, affected)
	})
}

// Unassign снимает выдачу с единицы и её состава, возвращая на хранение.
func (s *EquipmentService) Unassign(ctx context.Context, dto *models.UnassignEquipmentDTO) error {
	root, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if root.Status == models.EquipmentStatusWrittenOff {
		return models.ErrEquipmentNotAssignable
	}

	// Место возврата: явное из тела > «дом» единицы (место последнего периода без
	// держателя) > её текущее место. Возврат ставит единицу туда, откуда её выдали,
	// а не в кабинет держателя: место — реквизит единицы, с держателем оно не
	// приезжает и не уезжает. Дом ищем у корня и ставим всему составу — выдача тоже
	// пропагируется одним местом. Место всё же придётся указать (EQ008), если у
	// единицы его нет вообще: без держателя и без места она перестанет быть учтена,
	// а инвариант «держатель или место» держит CHECK в equipment_assignments.
	siteID, location := dto.SiteID, dto.Location
	if placeAbsent(siteID, location) {
		storedSiteID, storedLocation, err := s.repo.GetLastStoredPlace(ctx, dto.ID)
		switch {
		case err == nil:
			siteID, location = storedSiteID, storedLocation
		case errors.Is(err, models.ErrNotFound):
			siteID, location = root.SiteID, root.Location
		default:
			return fmt.Errorf("failed to get last stored place. error: %w", err)
		}
		if placeAbsent(siteID, location) {
			return models.ErrEquipmentNoHoldersOrLocation
		}
	}

	affected, err := s.collectSubtree(ctx, dto.RealmID, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to collect equipment subtree. error: %w", err)
	}

	// Название площадки возврата — тоже снимок события (см. resolveAssignmentLabels).
	_, siteName, err := s.resolveAssignmentLabels(ctx, nil, siteID)
	if err != nil {
		return err
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		opID := uuid.New()
		now := time.Now()
		actorID := actorID(dto.Actor)

		for _, eq := range affected {
			open, err := s.repo.GetOpenAssignment(ctx, eq.ID)
			if err != nil && !errors.Is(err, models.ErrNotFound) {
				return fmt.Errorf("failed to get open assignment. error: %w", err)
			}
			if err == nil {
				if err := s.repo.CloseAssignment(ctx, tx, open.ID, now, dto.Comment); err != nil {
					return fmt.Errorf("failed to close assignment. error: %w", err)
				}
			}

			if err := s.repo.CreateAssignment(ctx, tx, &models.EquipmentAssignment{
				EquipmentID: eq.ID,
				SiteID:      siteID,
				Location:    location,
				AssignedAt:  now,
				AssignedBy:  actorID,
				Comment:     dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create assignment. error: %w", err)
			}

			if err := s.repo.SetHolderAndStatus(ctx, tx, eq.ID, nil, nil, siteID, location, models.EquipmentStatusInStock); err != nil {
				return fmt.Errorf("failed to set holder. error: %w", err)
			}
			if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
				EquipmentID: eq.ID, EventType: models.EquipmentEventUnassigned, OperationID: opID,
				NewValue: assignmentJSON(&models.AssignEquipmentDTO{SiteID: siteID, Location: location}, "", siteName),
				UserID:   actorID, Comment: dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create event. error: %w", err)
			}
		}
		return nil
	})
}

// resolveAssignmentLabels достаёт подписи держателя и площадки для снимка события
// «выдано/возвращено»: в JSON рядом с id ложатся названия, чтобы лента истории не
// тянула справочники реалма, а снимок пережил удаление пользователя или площадки.
func (s *EquipmentService) resolveAssignmentLabels(ctx context.Context, userID, siteID *uuid.UUID) (string, string, error) {
	var holderName, siteName string

	if userID != nil {
		name, err := s.repo.GetUserFullName(ctx, *userID)
		if err != nil {
			return "", "", fmt.Errorf("failed to get holder name. error: %w", err)
		}
		holderName = name
	}
	if siteID != nil {
		name, err := s.repo.GetSiteName(ctx, *siteID)
		if err != nil {
			return "", "", fmt.Errorf("failed to get site name. error: %w", err)
		}
		siteName = name
	}
	return holderName, siteName, nil
}

// ChangeStatus меняет статус единицы. Списанную единицу менять нельзя.
func (s *EquipmentService) ChangeStatus(ctx context.Context, dto *models.ChangeStatusDTO) (*models.Equipment, error) {
	root, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if root.Status == models.EquipmentStatusWrittenOff {
		return nil, models.ErrEquipmentWrittenOff
	}

	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.SetStatus(ctx, tx, dto.RealmID, dto.ID, dto.Status); err != nil {
			return fmt.Errorf("failed to set status. error: %w", err)
		}
		eventType := models.EquipmentEventStatusChanged
		if dto.Status == models.EquipmentStatusWrittenOff {
			eventType = models.EquipmentEventWrittenOff
		}
		if dto.Status == models.EquipmentStatusRepair {
			eventType = models.EquipmentEventRepaired
		}
		return s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
			EquipmentID: dto.ID, EventType: eventType, OperationID: uuid.New(),
			OldValue: statusJSON(root.Status), NewValue: statusJSON(dto.Status),
			UserID: actorID(dto.Actor), Comment: dto.Comment,
		})
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
}

// GetAssignmentHistory возвращает историю выдач единицы.
func (s *EquipmentService) GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error) {
	data, err := s.repo.GetAssignmentHistory(ctx, realmID, equipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment history. error: %w", err)
	}
	return data, nil
}

// GetEvents возвращает журнал событий единицы.
func (s *EquipmentService) GetEvents(ctx context.Context, realmID, equipmentID uuid.UUID, limit, offset int) ([]*models.EquipmentEvent, error) {
	data, err := s.repo.GetEvents(ctx, realmID, equipmentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment events. error: %w", err)
	}
	return data, nil
}

// sameHolder сравнивает только держателя: место — реквизит единицы, поэтому его
// смена (место, например, исправляют вместе с выдачей) не делает выдачу «чужой».
// Строка-склад держателя не имеет, и её владельцем считается никто — конфликт
// считает conflictsWith, а не эта функция.
func sameHolder(open *models.EquipmentAssignment, dto *models.AssignEquipmentDTO) bool {
	if open == nil {
		return false
	}
	return sameUUID(open.UserID, dto.UserID) && sameString(open.HolderName, dto.HolderName)
}

// conflictsWith сообщает, что единица закреплена за другим держателем: только чужая
// выдача блокирует новую. Открытая строка без держателя (склад) конфликтом не
// является — после возврата такая строка остаётся, и без этой оговорки повторная
// выдача требовала бы force.
func conflictsWith(open *models.EquipmentAssignment, dto *models.AssignEquipmentDTO) bool {
	if open == nil || (open.UserID == nil && !hasText(open.HolderName)) {
		return false
	}
	return !sameHolder(open, dto)
}

// hasHolder сообщает, что в теле выдачи указан держатель — сотрудник или имя.
func hasHolder(dto *models.AssignEquipmentDTO) bool {
	return dto.UserID != nil || hasText(dto.HolderName)
}

// hasText отличает «поле не заполнено» от «поле пустое»: клиент шлёт nil, но
// свободный ввод может прийти пустой строкой или пробелами.
func hasText(s *string) bool {
	return s != nil && strings.TrimSpace(*s) != ""
}

// placeAbsent сообщает, что место не указано: и площадка, и локация пусты. Для
// выдачи это значит «единица не переезжает», для возврата — «ставь дом» (не
// «стереть место»): пустое поле означает отказ от изменения, а не сброс.
func placeAbsent(siteID *uuid.UUID, location *string) bool {
	return siteID == nil && !hasText(location)
}

func sameUUID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func sameString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// assignmentJSON сериализует держателя/локацию для old/new_value события. Кроме
// идентификаторов хранятся подписи (ФИО держателя, название площадки): лента истории
// не тянет справочники реалма, а снимок переживает удаление пользователя/площадки
// (FK объявлены ON DELETE SET NULL) и переименование.
func assignmentJSON(dto *models.AssignEquipmentDTO, holderFullName, siteName string) []byte {
	data, err := json.Marshal(map[string]any{
		"userId":     dto.UserID,
		"userName":   holderFullName,
		"holderName": dto.HolderName,
		"siteId":     dto.SiteID,
		"siteName":   siteName,
		"location":   dto.Location,
	})
	if err != nil {
		return nil
	}
	return data
}

// statusJSON сериализует статус для old/new_value события.
func statusJSON(status models.EquipmentStatus) []byte {
	data, err := json.Marshal(map[string]any{"status": status})
	if err != nil {
		return nil
	}
	return data
}

// relocationJSON сериализует место единицы для события «перемещено». В снимке и
// идентификатор, и название площадки: журнал иммутабелен, а площадку могут
// переименовать или удалить — подпись должна это пережить.
func relocationJSON(eq *models.Equipment) []byte {
	data, err := json.Marshal(map[string]any{
		"siteId":   eq.SiteID,
		"siteName": eq.SiteName,
		"location": eq.Location,
	})
	if err != nil {
		return nil
	}
	return data
}

// equipmentCreatedJSON сериализует снимок единицы для события «создано»: в
// истории достаточно опознавательных полей (карточка поступления на склад).
func equipmentCreatedJSON(eq *models.Equipment) []byte {
	data, err := json.Marshal(map[string]any{
		"name":            eq.Name,
		"serialNumber":    eq.SerialNumber,
		"inventoryNumber": eq.InventoryNumber,
	})
	if err != nil {
		return nil
	}
	return data
}

// equipmentRelocated сообщает, сменились ли площадка или локация единицы: только
// такие правки попадают в журнал как перемещение. Форма редактирования присылает
// все поля целиком, поэтому сравнивать нужно со состоянием до правки.
func equipmentRelocated(previous, updated *models.Equipment) bool {
	return !sameUUID(previous.SiteID, updated.SiteID) || !sameString(previous.Location, updated.Location)
}
