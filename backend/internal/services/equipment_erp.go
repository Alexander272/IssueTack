package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// erpOperation — вид операции, по которой заводится заявка в 1C:ERP.
type erpOperation string

const (
	erpAssign   erpOperation = "assign"
	erpUnassign erpOperation = "unassign"
)

// createERPTicket заводит заявку «провести передачу техники в 1C:ERP» после выдачи
// или возврата. Создатель, заказчик и исполнитель — тот, кто проводил операцию:
// заявка живёт в его правах, поэтому проходит обычный TicketService.Create
// (проверка права на создание, журнал, уведомления) — отдельного «системного» пути
// нет. Исполнитель задаётся явно: иначе autoAssign подставил бы ответственного по
// умолчанию группы, и заявку взял бы кто угодно, кроме проводившего операцию.
//
// Реквизиты заявки (категория/группа/площадка) задаются в конфиге: тикет без группы
// создать нельзя, а выдумывать группу или категорию нельзя тем более. Если хотя бы
// один реквизит пуст — заявка не создаётся, операция не считается неуспешной.
//
// Ошибка создания ERP-заявки не откатывает выдачу/возврат: те уже закоммичены к
// моменту вызова. Поэтому ошибка не возвращается наверх — иначе успешная выдача
// выглядела бы как неудавшаяся — а уходит через bestEffortError: в лог и в
// error_bot, чтобы факт сбоя не потерялся.
func (s *EquipmentService) createERPTicket(ctx context.Context, op erpOperation, id, realmID uuid.UUID, actor *models.Actor) {
	if s.tickets == nil || actor == nil {
		return
	}

	categoryID, err := uuid.Parse(s.erpConf.ERPCategoryID)
	if err != nil {
		return
	}
	groupID, err := uuid.Parse(s.erpConf.ERPGroupID)
	if err != nil {
		return
	}
	siteID, err := uuid.Parse(s.erpConf.ERPSiteID)
	if err != nil {
		return
	}

	// Реквизиты единицы читаем заново: выдача/возврат уже закоммичены, и держатель
	// в БД новый — по «старому» снимку описание заявки показывало бы прежнего.
	eq, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: id, RealmID: realmID})
	if err != nil {
		bestEffortError("failed to load equipment for 1C:ERP ticket", err, map[string]string{
			"equipment_id": id.String(),
		})
		return
	}

	title, description := erpTicketContent(op, eq)
	dto := &models.TicketDTO{
		Title:       title,
		Description: description,
		Status:      models.StatusOpen,
		Priority:    models.PriorityMedium,
		RealmID:     &eq.RealmID,
		SiteID:      siteID,
		CategoryID:  categoryID,
		GroupID:     &groupID,
		CreatorID:   actor.ID,
		// Заказчик и исполнитель — тот, кто выдал: заявка «для себя», и это надо
		// задать явно. Пустой AssigneeID уходит в autoAssign, и исполнителем
		// становится ответственный по умолчанию группы (или единственный
		// участник) — то есть заявку на 1C:ERP взял бы кто угодно, кроме того,
		// кто операцию и проводил. Ветка исполнителя (applyExecutorCreateRestrictions)
		// оставляет AssigneeID только равным CreatorID — у нас так и есть.
		OwnerID:     &actor.ID,
		AssigneeID:  &actor.ID,
		EquipmentID: &eq.ID,
		Actor:       actor,
	}

	if err := s.tickets.Create(ctx, dto); err != nil {
		bestEffortError("failed to create 1C:ERP ticket", err, map[string]string{
			"equipment_id": eq.ID.String(),
			"operation":    string(op),
		})
	}
}

// erpTicketContent собирает заголовок и описание ERP-заявки. Заголовок называет
// операцию, описание перечисляет реквизиты единицы, чтобы исполнителю не открывать
// карточку оборудования ради номера или держателя.
func erpTicketContent(op erpOperation, eq *models.Equipment) (string, string) {
	action := "передачу"
	if op == erpUnassign {
		action = "приёмку"
	}

	title := fmt.Sprintf("Провести %s техники в 1C:ERP: %s", action, eq.Name)

	holder := eq.HolderFullName
	if holder == "" && eq.HolderName != nil {
		holder = *eq.HolderName
	}

	description := fmt.Sprintf("Оборудование: %s", eq.Name)
	if eq.InventoryNumber != nil && *eq.InventoryNumber != "" {
		description += fmt.Sprintf("\nИнвентарный номер: %s", *eq.InventoryNumber)
	}
	if eq.SerialNumber != nil && *eq.SerialNumber != "" {
		description += fmt.Sprintf("\nСерийный номер: %s", *eq.SerialNumber)
	}
	if holder != "" {
		description += fmt.Sprintf("\nДержатель: %s", holder)
	}
	if eq.SiteName != "" {
		description += fmt.Sprintf("\nПлощадка: %s", eq.SiteName)
	}
	if op == erpAssign {
		description += "\n\nПровести выдачу техники в 1C:ERP."
	} else {
		description += "\n\nПровести приёмку техники в 1C:ERP."
	}

	return title, description
}
