package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// equipmentReturnReminder — напоминание о сроке возврата выданного оборудования.
// Живёт отдельно от NotificationService: тот построен вокруг агрегата заявки
// (Notifier получает *models.Ticket, дедупликация — по ticket_id), а здесь предмет
// напоминания — выдача, и адресат ровно один: тот, кто выдал. Поэтому вместо
// каналов уведомлений используется тот же Mattermost-отправитель напрямую.
type equipmentReturnReminder struct {
	equipment EquipmentDueReturns
	mmRepo    repository.Mattermost
	users     Users
	sender    mattermostSender
}

// EquipmentDueReturns — часть репозитория оборудования, нужная напоминанию:
// открытые выдачи с наступившим сроком возврата.
type EquipmentDueReturns interface {
	// GetDueReturns возвращает открытые выдачи со сроком возврата не позже now.
	GetDueReturns(ctx context.Context, now time.Time) ([]*models.EquipmentDueReturn, error)
}

// EquipmentReturnReminderDeps содержит зависимости напоминания о сроке возврата.
type EquipmentReturnReminderDeps struct {
	Equipment EquipmentDueReturns
	MMRepo    repository.Mattermost
	Users     Users
	Sender    mattermostSender
}

// newEquipmentReturnReminder создаёт напоминание о сроке возврата. Любая из
// зависимостей может быть nil (например, интеграция Mattermost не настроена) —
// тогда рассылка молча ничего не делает.
func newEquipmentReturnReminder(deps *EquipmentReturnReminderDeps) *equipmentReturnReminder {
	return &equipmentReturnReminder{
		equipment: deps.Equipment,
		mmRepo:    deps.MMRepo,
		users:     deps.Users,
		sender:    deps.Sender,
	}
}

// NotifyDueReturns рассылает напоминания о наступившем сроке возврата. Напоминание
// уходит тому, кто выдал единицу. Ошибка по одной выдаче не прерывает остальные;
// возвращается число фактически отправленных сообщений.
func (r *equipmentReturnReminder) NotifyDueReturns(ctx context.Context, now time.Time) (int, error) {
	if r.equipment == nil || r.mmRepo == nil || r.users == nil || r.sender == nil {
		return 0, nil
	}

	due, err := r.equipment.GetDueReturns(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("failed to get due returns. error: %w", err)
	}

	sent := 0
	for _, item := range due {
		ok, err := r.notifyIssuer(ctx, item)
		if err != nil {
			logger.Warn("failed to send equipment return reminder",
				logger.StringAttr("equipment_id", item.EquipmentID.String()),
				logger.StringAttr("issued_by", item.IssuedBy.String()),
				logger.ErrAttr(err),
			)
			continue
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

// notifyIssuer отправляет DM тому, кто выдал единицу. Причина, по которой сообщение
// не отправлено (нет адреса/интеграции, пользователь не найден), — штатная, это не
// ошибка; ошибкой считается только сбой самой отправки.
func (r *equipmentReturnReminder) notifyIssuer(ctx context.Context, item *models.EquipmentDueReturn) (bool, error) {
	// Реалм берём из единицы: напоминание должно идти по интеграции того реалма, где
	// выдавали, а не по случайному домену. Настройки реалма подтягивает вызывающий
	// через саму единицу — здесь достаточно токена бота.
	settings, err := r.mmSettings(ctx, item)
	if err != nil {
		return false, err
	}
	if !settings.IsActive || settings.BotToken == "" {
		return false, nil
	}

	user, err := r.users.GetByID(ctx, item.IssuedBy)
	if err != nil {
		logger.Warn("failed to load user for equipment return reminder",
			logger.StringAttr("user_id", item.IssuedBy.String()),
			logger.ErrAttr(err),
		)
		return false, nil
	}
	if user.MattermostID == nil || *user.MattermostID == "" {
		return false, nil
	}

	message := r.format(item)
	if err := r.sender.Send(settings.BotToken, settings.BotUserID, *user.MattermostID, message); err != nil {
		return false, fmt.Errorf("failed to send mattermost notification: %w", err)
	}
	logger.Info("equipment return reminder sent",
		logger.StringAttr("equipment_id", item.EquipmentID.String()),
		logger.StringAttr("user_id", item.IssuedBy.String()),
		logger.StringAttr("username", user.Username),
	)
	return true, nil
}

// mmSettings возвращает настройки Mattermost реалма единицы.
func (r *equipmentReturnReminder) mmSettings(ctx context.Context, item *models.EquipmentDueReturn) (*models.RealmMattermost, error) {
	if item.RealmID == uuid.Nil {
		// Реалм неизвестен — интеграцию не определить. Для выборки это не ошибка:
		// просто пропускаем напоминание.
		return &models.RealmMattermost{}, nil
	}
	settings, err := r.mmRepo.GetByRealm(ctx, item.RealmID)
	if err != nil {
		return nil, fmt.Errorf("failed to load mattermost settings. error: %w", err)
	}
	return settings, nil
}

// format собирает текст напоминания о сроке возврата. Показывает, кто держит
// единицу и где, чтобы выдавшему было понятно, к кому идти.
func (r *equipmentReturnReminder) format(item *models.EquipmentDueReturn) string {
	subject := item.Name
	if item.TypeName != "" {
		subject = fmt.Sprintf("%s (%s)", item.Name, item.TypeName)
	}
	if item.InventoryNumber != nil && *item.InventoryNumber != "" {
		subject = fmt.Sprintf("%s №%s", subject, *item.InventoryNumber)
	}

	holder := item.HolderName
	if holder == "" {
		holder = "не указан"
	}
	if item.SiteName != "" {
		holder = fmt.Sprintf("%s, %s", holder, item.SiteName)
	}

	return fmt.Sprintf("Пора вернуть оборудование\n%s\nДержатель: %s\nСрок возврата: %s",
		subject, holder, item.ReturnDueAt.Format("02.01.2006"))
}
