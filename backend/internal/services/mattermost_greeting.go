package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/assets"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
)

// greetingHeaderFile — имя файла скриншота в Mattermost (видно пользователю).
const greetingHeaderFile = "issue-track-create-ticket.png"

// greetingMessage — приветствие бота IT-поддержки. Отправляется один раз за
// пользователя в реалме: либо когда он сам создаёт личный диалог с ботом
// (событие direct_added), либо когда открывает диалог в webapp-плагине (иконка
// в шапке канала). Подробности — в AGENTS.md, раздел «Приветственное сообщение».
const greetingMessage = `👋 Привет! На связи чат-бот IT-поддержки.
Я помогу вам создать заявку в службу IT и зафиксировать техническую проблему, с которой вы столкнулись, а также отследить статус ваших обращений прямо здесь, в Mattermost.

💻 Если вы с компьютера (ПК):
Используй кнопку «Создать заявку» в шапке Mattermost (см. скриншот ниже 👇). Заполни форму и прикрепи любые файлы, скриншоты или логи к своему обращению. 

📱 Если вы с мобильного телефона:
Просто напиши мне любое слово в этот чат (например, привет или помощь), и я пришлю меню с кнопками и полным списком команд.`

// PluginGreeting отправляет приветствие пользователю, открывшему личный диалог
// с ботом реалма. Вызывается webapp-плагином при монтировании иконки в шапке
// канала; идемпотентно — повторные вызовы ничего не шлют. Непривязанный канал —
// не ошибка: приветствия просто нет.
func (s *MattermostService) PluginGreeting(ctx context.Context, scope models.PluginScope) error {
	settings, err := s.resolvePluginSettings(ctx, scope)
	if err != nil {
		return nil
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, scope.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	return s.sendGreeting(ctx, settings, user.ID, scope.ChannelID)
}

// sendGreeting доставляет приветствие в личный диалог один раз за пользователя.
// Пометка «уже приветствовали» ставится до отправки (ClaimGreeting): это
// защищает от гонки, когда событие direct_added и открытие диалога из плагина
// приходят почти одновременно. Если отправка не удалась, пометка снимается —
// следующий триггер повторит попытку.
func (s *MattermostService) sendGreeting(ctx context.Context, settings *models.RealmMattermost, userID uuid.UUID, channelID string) error {
	if s.most == nil || s.most.Client == nil {
		return models.ErrChannelNotBound
	}
	if channelID == "" {
		return nil
	}

	claimed, err := s.repo.ClaimGreeting(ctx, settings.RealmID, userID)
	if err != nil {
		return fmt.Errorf("failed to claim greeting: %w", err)
	}
	if !claimed {
		return nil
	}

	if err := s.postGreeting(settings, channelID); err != nil {
		if relErr := s.repo.ReleaseGreeting(ctx, settings.RealmID, userID); relErr != nil {
			logger.Warn("failed to release greeting claim",
				logger.StringAttr("user_id", userID.String()),
				logger.StringAttr("realm_id", settings.RealmID.String()),
				logger.ErrAttr(relErr))
		}
		return err
	}

	logger.Info("mattermost greeting sent",
		logger.StringAttr("user_id", userID.String()),
		logger.StringAttr("realm_id", settings.RealmID.String()))
	return nil
}

// postGreeting собирает и отправляет само сообщение. Скриншот кнопки —
// best-effort: если Mattermost не принял файл, приветствие всё равно уходит
// текстом (важнее сам текст, чем картинка).
func (s *MattermostService) postGreeting(settings *models.RealmMattermost, channelID string) error {
	var fileIDs []string
	if len(assets.HeaderPNG) > 0 {
		info, err := s.most.Client.UploadFile(settings.BotToken, channelID, assets.HeaderPNG, greetingHeaderFile)
		if err != nil {
			logger.Warn("failed to upload greeting image", logger.ErrAttr(err))
		} else {
			fileIDs = []string{info.Id}
		}
	}

	if _, err := s.most.Post.Create(settings.BotToken, mattermost.CreatePostDTO{
		ChannelID: channelID,
		Message:   greetingMessage,
		FileIDs:   fileIDs,
	}); err != nil {
		return fmt.Errorf("failed to send greeting: %w", err)
	}
	return nil
}
