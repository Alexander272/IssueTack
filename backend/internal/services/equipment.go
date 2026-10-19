package services

import (
	"context"
	"fmt"
	"io"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// EquipmentService — сервис учёта оборудования (справочник типов и единицы).
type EquipmentService struct {
	repo repository.Equipment
	tm   TransactionManager
	// conf — каталог и лимит загрузок: подписанные акты лежат рядом с вложениями
	// заявок, поэтому берут те же настройки файлового хранилища.
	conf *config.FileServerConfig
}

// NewEquipmentService создаёт EquipmentService.
func NewEquipmentService(repo repository.Equipment, tm TransactionManager, conf *config.FileServerConfig) *EquipmentService {
	return &EquipmentService{repo: repo, tm: tm, conf: conf}
}

// Equipment — интерфейс работы с оборудованием.
type Equipment interface {
	// GetTypes возвращает список типов оборудования реалма.
	GetTypes(ctx context.Context, req *models.GetEquipmentTypesDTO) ([]*models.EquipmentType, error)
	// GetTypeByID возвращает тип оборудования по идентификатору.
	GetTypeByID(ctx context.Context, req *models.GetEquipmentTypeByIdDTO) (*models.EquipmentType, error)
	// CreateType создаёт тип оборудования.
	CreateType(ctx context.Context, dto *models.EquipmentTypeDTO) error
	// UpdateType обновляет тип оборудования.
	UpdateType(ctx context.Context, dto *models.EquipmentTypeDTO) error
	// DeleteType удаляет тип оборудования.
	DeleteType(ctx context.Context, dto *models.DelEquipmentTypeDTO) error

	// Get возвращает список единиц оборудования реалма с фильтрами и пагинацией.
	Get(ctx context.Context, filter *models.EquipmentFilter) ([]*models.Equipment, int, error)
	// GetByID возвращает единицу оборудования.
	GetByID(ctx context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error)
	// GetChildren возвращает состав комплекта (непосредственных потомков).
	GetChildren(ctx context.Context, realmID, parentID uuid.UUID) ([]*models.Equipment, error)
	// GetTree возвращает один уровень дерева оборудования реалма: корни (ParentID
	// == nil) либо детей узла. Второе значение — всего на уровне (для пагинации).
	GetTree(ctx context.Context, dto *models.GetEquipmentTreeDTO) ([]*models.EquipmentTreeNode, int, error)
	// GetStats возвращает счётчики единиц реалма для шапки страницы.
	GetStats(ctx context.Context, realmID uuid.UUID) (*models.EquipmentStats, error)
	// Create создаёт единицу оборудования (для комплекта — сразу с составом).
	Create(ctx context.Context, dto *models.CreateEquipmentDTO) (*models.Equipment, error)
	// Update обновляет единицу оборудования (с проверкой циклов в дереве комплектов).
	Update(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error)
	// Delete удаляет единицу оборудования.
	Delete(ctx context.Context, dto *models.DelEquipmentDTO) error

	// Assign выдаёт единицу (или комплект) держателю, пропагируя на состав.
	Assign(ctx context.Context, dto *models.AssignEquipmentDTO) error
	// Unassign возвращает единицу на хранение (снимает выдачу).
	Unassign(ctx context.Context, dto *models.UnassignEquipmentDTO) error
	// ChangeStatus меняет статус единицы оборудования.
	ChangeStatus(ctx context.Context, dto *models.ChangeStatusDTO) (*models.Equipment, error)
	// GetAssignmentHistory возвращает историю выдач единицы.
	GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error)
	// GetEvents возвращает журнал событий единицы.
	GetEvents(ctx context.Context, realmID, equipmentID uuid.UUID, limit, offset int) ([]*models.EquipmentEvent, error)

	// GetComposition возвращает текущий состав логического комплекта.
	GetComposition(ctx context.Context, realmID, kitID uuid.UUID) ([]*models.Equipment, error)
	// GetCompositionHistory возвращает историю состава комплекта.
	GetCompositionHistory(ctx context.Context, realmID, kitID uuid.UUID) ([]*models.KitComposition, error)
	// AddComponents добавляет единицы в состав комплекта.
	AddComponents(ctx context.Context, dto *models.KitCompositionOpDTO) error
	// RemoveComponents исключает единицы из состава комплекта.
	RemoveComponents(ctx context.Context, dto *models.KitCompositionOpDTO) error

	// GetKitTemplates возвращает шаблоны комплектов реалма вместе с позициями.
	GetKitTemplates(ctx context.Context, realmID uuid.UUID) ([]*models.EquipmentKitTemplate, error)
	// GetKitTemplateByID возвращает шаблон комплекта по идентификатору.
	GetKitTemplateByID(ctx context.Context, req *models.GetEquipmentKitTemplateByIdDTO) (*models.EquipmentKitTemplate, error)
	// CreateKitTemplate создаёт шаблон комплекта.
	CreateKitTemplate(ctx context.Context, dto *models.EquipmentKitTemplateDTO) error
	// UpdateKitTemplate обновляет шаблон комплекта (состав заменяется целиком).
	UpdateKitTemplate(ctx context.Context, dto *models.EquipmentKitTemplateDTO) error
	// DeleteKitTemplate удаляет шаблон комплекта.
	DeleteKitTemplate(ctx context.Context, dto *models.DelEquipmentKitTemplateDTO) error

	// GetActs возвращает список актов реалма.
	GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error)
	// GetActByID возвращает акт по идентификатору.
	GetActByID(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, error)
	// CreateAct формирует акт вручную.
	CreateAct(ctx context.Context, dto *models.EquipmentActDTO) (*models.EquipmentAct, error)
	// GetActDocument собирает docx акта по снимку состава.
	GetActDocument(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentActDocument, error)
	// SignAct сохраняет подписанный акт и переводит его в статус «подписан».
	SignAct(ctx context.Context, dto *models.SignEquipmentActDTO) error
	// GetActSignedFile отдаёт подписанный файл акта.
	GetActSignedFile(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, io.ReadCloser, error)
}

// Get возвращает список единиц оборудования реалма с фильтрами и пагинацией.
func (s *EquipmentService) Get(ctx context.Context, filter *models.EquipmentFilter) ([]*models.Equipment, int, error) {
	data, total, err := s.repo.Get(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get equipment. error: %w", err)
	}
	return data, total, nil
}

// GetByID возвращает единицу оборудования.
func (s *EquipmentService) GetByID(ctx context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment by id. error: %w", err)
	}
	return data, nil
}

// GetChildren возвращает состав комплекта (непосредственных потомков).
func (s *EquipmentService) GetChildren(ctx context.Context, realmID, parentID uuid.UUID) ([]*models.Equipment, error) {
	data, err := s.repo.GetChildren(ctx, realmID, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment children. error: %w", err)
	}
	return data, nil
}

// GetTree возвращает один уровень дерева оборудования реалма.
func (s *EquipmentService) GetTree(ctx context.Context, dto *models.GetEquipmentTreeDTO) ([]*models.EquipmentTreeNode, int, error) {
	data, total, err := s.repo.GetTree(ctx, dto)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get equipment tree. error: %w", err)
	}
	return data, total, nil
}

// GetStats возвращает счётчики единиц реалма для шапки страницы.
func (s *EquipmentService) GetStats(ctx context.Context, realmID uuid.UUID) (*models.EquipmentStats, error) {
	data, err := s.repo.GetStats(ctx, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment stats. error: %w", err)
	}
	return data, nil
}

// Create создаёт единицу оборудования. Вставка и событие журнала «создано» идут
// одной транзакцией: журнал — часть операции, а не пост-обработка. Для комплекта
// можно сразу передать стартовый состав — он открывается в той же транзакции,
// поэтому «комплект без состава» при частичном сбое не остаётся.
func (s *EquipmentService) Create(ctx context.Context, dto *models.CreateEquipmentDTO) (*models.Equipment, error) {
	eq := &dto.EquipmentDTO
	normalizeEquipmentDTO(eq)
	deriveHolderStatus(eq)

	// У комплекта типа нет: присланный игнорируем. Единице тип обязателен —
	// колонка nullable только потому, что одно поле обслуживает оба случая.
	if eq.IsKit {
		eq.TypeID = nil
	} else if eq.TypeID == nil {
		return nil, models.ErrInvalidInput
	}

	if len(dto.Components) > 0 && !eq.IsKit {
		return nil, models.ErrEquipmentNotKit
	}

	if err := s.ensureParentCanBePart(ctx, eq); err != nil {
		return nil, err
	}
	if err := s.ensureParentCanHost(ctx, eq); err != nil {
		return nil, err
	}

	var data *models.Equipment
	err := s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if eq.InventoryNumber == nil {
			number, err := s.nextInventoryNumber(ctx, tx, eq.RealmID, inventorySeries(eq.IsKit))
			if err != nil {
				return err
			}
			eq.InventoryNumber = &number
		}

		created, err := s.repo.Create(ctx, tx, eq)
		if err != nil {
			return fmt.Errorf("failed to create equipment. error: %w", err)
		}
		data = created
		if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
			EquipmentID: created.ID, EventType: models.EquipmentEventCreated, OperationID: uuid.New(),
			NewValue: equipmentCreatedJSON(created), UserID: actorID(eq.Actor),
		}); err != nil {
			return err
		}
		if len(dto.Components) == 0 {
			return nil
		}
		return s.openComponents(ctx, tx, &models.OpenComponentsDTO{
			RealmID: eq.RealmID, KitID: created.ID, Components: dto.Components,
			Reason: "Создание комплекта", Actor: eq.Actor,
		})
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Update обновляет единицу оборудования. Смена родителя проверяется на цикл:
// новый родитель не может быть потомком (или самим) обновляемой единицы.
// Изменение площадки или локации отдельной записью попадает в журнал как
// «перемещено» — иначе перемещение было бы видно только в выдаче/возврате.
func (s *EquipmentService) Update(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error) {
	if dto.ID == nil {
		return nil, models.ErrInvalidInput
	}
	normalizeEquipmentDTO(dto)
	deriveHolderStatus(dto)

	// У комплекта типа нет — правка формы не должна его ему навязывать.
	if dto.IsKit {
		dto.TypeID = nil
	} else if dto.TypeID == nil {
		return nil, models.ErrInvalidInput
	}

	if err := s.ensureParentCanBePart(ctx, dto); err != nil {
		return nil, err
	}
	if err := s.ensureParentCanHost(ctx, dto); err != nil {
		return nil, err
	}

	if dto.ParentID != nil {
		if *dto.ParentID == *dto.ID {
			return nil, models.ErrEquipmentCycle
		}
		ancestors, err := s.repo.GetAncestors(ctx, dto.RealmID, *dto.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get equipment ancestors. error: %w", err)
		}
		for _, ancestor := range ancestors {
			if ancestor == *dto.ID {
				return nil, models.ErrEquipmentCycle
			}
		}
	}

	// Состояние до правки нужно, чтобы отличить перемещение от обычного
	// сохранения формы: она присылает все поля целиком.
	previous, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: *dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment. error: %w", err)
	}

	var data *models.Equipment
	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if dto.InventoryNumber == nil {
			// Серию берём из состояния до правки: `is_kit` репозиторий не
			// перезаписывает, то есть комплект комплектом и остаётся, а форма
			// могла не прислать флаг.
			number, err := s.nextInventoryNumber(ctx, tx, dto.RealmID, inventorySeries(previous.IsKit))
			if err != nil {
				return err
			}
			dto.InventoryNumber = &number
		}

		updated, err := s.repo.Update(ctx, tx, dto)
		if err != nil {
			return fmt.Errorf("failed to update equipment. error: %w", err)
		}
		data = updated
		if !equipmentRelocated(previous, updated) {
			return nil
		}
		return s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
			EquipmentID: updated.ID, EventType: models.EquipmentEventMoved, OperationID: uuid.New(),
			OldValue: relocationJSON(previous),
			NewValue: relocationJSON(updated),
			UserID:   actorID(dto.Actor),
		})
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Delete удаляет единицу оборудования. Потомки (если были) автоматически
// отсоединяются от комплекта (parent_id = NULL).
func (s *EquipmentService) Delete(ctx context.Context, dto *models.DelEquipmentDTO) error {
	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete equipment. error: %w", err)
	}
	return nil
}

// ensureParentCanBePart проверяет, что единица, у которой задан родитель
// (физический состав, parent_id), относится к типу, допустимому в составе:
// комплектующему или расходнику. Комплекту parent_id не нужен — состав ведётся
// через kit_compositions, поэтому комплект в чужой физический состав не кладётся.
func (s *EquipmentService) ensureParentCanBePart(ctx context.Context, dto *models.EquipmentDTO) error {
	if dto.ParentID == nil {
		return nil
	}
	// Типа нет только у комплекта — он в физический состав не входит.
	if dto.IsKit || dto.TypeID == nil {
		return models.ErrEquipmentParentNotPart
	}
	eqType, err := s.repo.GetTypeByID(ctx, &models.GetEquipmentTypeByIdDTO{ID: *dto.TypeID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment type. error: %w", err)
	}
	if !eqType.Kind.CanBePart() {
		return models.ErrEquipmentParentNotPart
	}
	return nil
}

// ensureParentCanHost проверяет, что единица, которой назначают родителя
// (физический состав, parent_id), способна содержать состав: только устройство
// или периферия. В комплектующее или расходник физически ничего вложить нельзя,
// а комплект ведёт состав логически — через kit_compositions, не через parent_id.
func (s *EquipmentService) ensureParentCanHost(ctx context.Context, dto *models.EquipmentDTO) error {
	if dto.ParentID == nil {
		return nil
	}
	parent, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: *dto.ParentID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get parent equipment. error: %w", err)
	}
	if parent.TypeKind.CanBePart() || parent.IsKit {
		return models.ErrEquipmentParentCannotHost
	}
	return nil
}

// normalizeEquipmentDTO подчищает опциональные строки: пустые -> NULL (иначе
// partial unique-индексы и CHECK работали бы с ”) и specs без значения -> {}.
func normalizeEquipmentDTO(dto *models.EquipmentDTO) {
	emptyToNil := func(v *string) *string {
		if v != nil && *v == "" {
			return nil
		}
		return v
	}
	dto.SerialNumber = emptyToNil(dto.SerialNumber)
	dto.InventoryNumber = emptyToNil(dto.InventoryNumber)
	dto.HolderName = emptyToNil(dto.HolderName)
	dto.Location = emptyToNil(dto.Location)
}

// inventorySeries выбирает серию автоинвентарного номера: комплект нумеруется
// отдельно от единиц (с префиксом «KIT-»), чтобы значения двух серий не путались.
func inventorySeries(isKit bool) models.EquipmentInventorySeries {
	if isKit {
		return models.InventorySeriesKit
	}
	return models.InventorySeriesUnit
}

// nextInventoryNumber выдаёт свободный инвентарный номер серии в реалме. Счётчик
// монотонный, но ручной ввод (или перенос из старой системы) мог занять будущее
// значение, поэтому кандидат проверяется на занятость и счётчик листается дальше.
// Проверка — обычный SELECT, а не попытка вставки: упавший statement абортит
// транзакцию, и повторить в ней вставку уже нельзя. Лимит попыток — предохранитель
// от зацикливания, на практике не достигается.
func (s *EquipmentService) nextInventoryNumber(ctx context.Context, tx postgres.Tx, realmID uuid.UUID, series models.EquipmentInventorySeries) (string, error) {
	const maxAttempts = 50

	for i := 0; i < maxAttempts; i++ {
		seq, err := s.repo.NextInventoryNumber(ctx, tx, realmID, series)
		if err != nil {
			return "", fmt.Errorf("failed to get inventory number counter. error: %w", err)
		}
		number := models.InventoryNumber(series, seq)
		taken, err := s.repo.InventoryNumberExists(ctx, tx, realmID, number)
		if err != nil {
			return "", fmt.Errorf("failed to check inventory number. error: %w", err)
		}
		if !taken {
			return number, nil
		}
	}
	return "", fmt.Errorf("failed to generate inventory number for realm %s: no free value in %d attempts", realmID, maxAttempts)
}

// deriveHolderStatus выводит пару «на складе»/«в эксплуатации» из держателя и
// вызывается после normalizeEquipmentDTO (пустой holderName должен стать nil).
//
// Эти два статуса — не самостоятельное состояние единицы, а следствие выдачи:
// Assign/Unassign проставляют их вместе с держателем (SetHolderAndStatus). Форма
// же присылает статус и держателя независимо, и без вывода получалась единица
// «в эксплуатации» без держателя — блок «Держатель» показывал на неё «На складе»,
// хотя единица никому не выдана. Обратный случай (склад при заполненном
// держателе) так же невозможен: выданная единица уже не лежит на складе.
//
// Остальные статусы (ремонт, дефект, резерв, списание) задаются вручную и от
// держателя не зависят: дефектная единица может быть на руках.
func deriveHolderStatus(dto *models.EquipmentDTO) {
	if dto.Status != models.EquipmentStatusInStock && dto.Status != models.EquipmentStatusInUse {
		return
	}
	if dto.HolderUserID != nil || (dto.HolderName != nil && *dto.HolderName != "") {
		dto.Status = models.EquipmentStatusInUse
		return
	}
	dto.Status = models.EquipmentStatusInStock
}

// collectSubtree собирает единицу и весь её состав: физических потомков
// (parent_id) и, для комплектов, текущих логических участников (kit_compositions).
func (s *EquipmentService) collectSubtree(ctx context.Context, realmID, rootID uuid.UUID) ([]*models.Equipment, error) {
	seen := map[uuid.UUID]bool{}
	result := []*models.Equipment{}
	queue := []uuid.UUID{rootID}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true

		item, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: id, RealmID: realmID})
		if err != nil {
			return nil, err
		}
		result = append(result, item)

		children, err := s.repo.GetChildren(ctx, realmID, id)
		if err != nil {
			return nil, err
		}
		for _, c := range children {
			queue = append(queue, c.ID)
		}
		if item.IsKit {
			members, err := s.repo.GetOpenComposition(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				queue = append(queue, m.ID)
			}
		}
	}
	return result, nil
}

func actorID(a *models.Actor) *uuid.UUID {
	if a == nil || a.ID == uuid.Nil {
		return nil
	}
	id := a.ID
	return &id
}
