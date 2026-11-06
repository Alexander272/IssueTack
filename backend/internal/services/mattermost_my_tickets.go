package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
)

const (
	// myTicketsChunkSize — сколько заявок помещается в одно сообщение бота:
	// список разбивается на сообщения, а не обрезается (молчаливый обрез до 20
	// выглядел для пользователя как «бот потерял заявки»). Карточка несёт
	// описание заявки, поэтому в сообщение влезает меньше заявок.
	myTicketsChunkSize = 5
	// myTicketsMaxCount — предохранитель: список длиннее отдаётся первыми
	// myTicketsMaxCount заявками, чтобы не завалить диалог сотнями постов.
	myTicketsMaxCount = 200
	// myTicketsPostDelay — пауза между сообщениями: Mattermost ограничивает
	// частоту постов, и серия сообщений без пауз частично отбрасывается с 429.
	myTicketsPostDelay = 350 * time.Millisecond
)

// myTicketAction* — действия кнопок в списке «Мои заявки». Правила переходов
// те же, что в плагине (см. PluginChangeStatus).
const (
	actionCancel  = "ticket_cancel"
	actionConfirm = "ticket_confirm"
	actionReopen  = "ticket_reopen"
	// reopenCallbackPrefix — префикс callback_id диалога «Вернуть в работу».
	reopenCallbackPrefix = actionReopen + ":"
	// cancelCallbackPrefix и confirmCallbackPrefix — префиксы callback_id
	// диалогов подтверждения необратимых переходов: отмены и закрытия заявки.
	cancelCallbackPrefix  = actionCancel + ":"
	confirmCallbackPrefix = actionConfirm + ":"
)

// actionTargetStatus — статус, в который переводит заявку действие из списка
// «Мои заявки». Целевой статус берём из самого действия, а не из контекста
// кнопки: контекст приходит от клиента и не должен решать, что делать.
func actionTargetStatus(action string) models.TicketStatus {
	switch action {
	case actionCancel:
		return models.StatusCancelled
	case actionConfirm:
		return models.StatusClosed
	case actionReopen:
		return models.StatusInProgress
	}
	return ""
}

// mmStatusLabels — подписи статусов заявки в карточках бота. Взяты из
// mmPlugin/webapp/src/labels.ts, чтобы список в Mattermost и в плагине называл
// статусы одинаково.
var mmStatusLabels = map[models.TicketStatus]string{
	models.StatusOpen:       "Новая",
	models.StatusInProgress: "В работе",
	models.StatusPending:    "Ожидание",
	models.StatusOnHold:     "Отложена",
	models.StatusResolved:   "Решена",
	models.StatusClosed:     "Закрыта",
	models.StatusCancelled:  "Отменена",
}

// mmStatusColors — цвет полосы карточки по статусу (значения из labels.ts плагина).
var mmStatusColors = map[models.TicketStatus]string{
	models.StatusOpen:       "#01579B",
	models.StatusInProgress: "#E65100",
	models.StatusPending:    "#F57F17",
	models.StatusOnHold:     "#4A148C",
	models.StatusResolved:   "#1B5E20",
	models.StatusClosed:     "#024A02",
	models.StatusCancelled:  "#B71C1C",
}

// myActiveTickets собирает заявки, которые показываем пользователю по кнопке
// «Мои заявки» и по команде «мои»: активные, где он автор ИЛИ заказчик —
// тот же набор, что у вкладки «Мои заявки» в плагине, чтобы список не расходился
// между клиентами. Resolved оставлен, чтобы по таким заявкам можно было
// подтвердить решение или вернуть в работу.
func (s *MattermostService) myActiveTickets(ctx context.Context, realmID uuid.UUID, user *models.UserData) ([]*models.Ticket, error) {
	mode := "created_or_owned"
	tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		RealmID: &realmID,
		Statuses: []models.TicketStatus{
			models.StatusOpen,
			models.StatusInProgress,
			models.StatusPending,
			models.StatusOnHold,
			models.StatusResolved,
		},
		Actor:            &models.Actor{ID: user.ID, Name: user.Username},
		Mode:             &mode,
		CreatedOrOwnedBy: &user.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user tickets: %w", err)
	}
	return tickets, nil
}

// myTicketsChunk — одно сообщение со списком заявок пользователя: заголовок
// и карточки с действиями.
type myTicketsChunk struct {
	message string
	cards   []mattermost.Attachment
}

// myTicketsChunks разбивает заявки пользователя на сообщения по
// myTicketsChunkSize — показываем все, а не первые 20.
func (s *MattermostService) myTicketsChunks(tickets []*models.Ticket, user *models.UserData) []myTicketsChunk {
	if len(tickets) == 0 {
		return []myTicketsChunk{{message: "У вас нет активных заявок."}}
	}
	if len(tickets) > myTicketsMaxCount {
		tickets = tickets[:myTicketsMaxCount]
	}

	chunks := make([]myTicketsChunk, 0, (len(tickets)+myTicketsChunkSize-1)/myTicketsChunkSize)
	for from := 0; from < len(tickets); from += myTicketsChunkSize {
		chunk, ok := s.myTicketsChunkAt(tickets, from, user)
		if !ok {
			break
		}
		chunks = append(chunks, chunk)
	}
	return chunks
}

// myTicketsChunkAt собирает одно сообщение со срезом заявок [from, from+размер).
// Смещение from кладут в контекст кнопок карточек: после смены статуса бот
// перерисовывает тем же смещением тот же срез.
func (s *MattermostService) myTicketsChunkAt(tickets []*models.Ticket, from int, user *models.UserData) (myTicketsChunk, bool) {
	if from < 0 || from >= len(tickets) {
		return myTicketsChunk{}, false
	}
	end := min(from+myTicketsChunkSize, len(tickets))
	return s.myTicketsChunk(tickets, from, end, myTicketsHeader(from, end, len(tickets)), user), true
}

// myTicketsChunkAfterEdit пересобирает сообщение списка после действия над
// заявкой, нажатой в срезе [from, ...).
//
//	removed=true  — заявка ушла из активного списка (отмена, подтверждение
//	                решения): показываем ровно тот же срез без этой карточки.
//	                Всё, что было ниже удалённой заявки, сдвинулось на один
//	                элемент, поэтому срез нужно взять на элемент короче, иначе
//	                в него подтянется лишняя карточка из следующего сообщения.
//	removed=false — заявка осталась в списке (возврат в работу): перерисовываем
//	                срез целиком, карточка получит новый статус и кнопки.
//
// Заголовок всегда считается по состоянию ДО действия, поэтому он не «поедет»
// после удаления карточки. ok=false означает, что в сообщении не осталось ни
// одной карточки — такое сообщение надо удалить.
func (s *MattermostService) myTicketsChunkAfterEdit(tickets []*models.Ticket, from int, removed bool, user *models.UserData) (myTicketsChunk, bool) {
	if from < 0 || from > len(tickets) {
		return myTicketsChunk{}, false
	}

	total := len(tickets)
	if removed {
		total++ // убираем из тотала заявку, которая выпала из списка
	}
	oldEnd := min(from+myTicketsChunkSize, total)

	end := oldEnd
	if removed {
		end--
	}
	end = min(end, len(tickets))
	if from >= end {
		return myTicketsChunk{}, false
	}
	return s.myTicketsChunk(tickets, from, end, myTicketsHeader(from, oldEnd, total), user), true
}

// myTicketsChunk превращает срез заявок в карточки поста с готовым заголовком.
func (s *MattermostService) myTicketsChunk(tickets []*models.Ticket, from, end int, message string, user *models.UserData) myTicketsChunk {
	cards := make([]mattermost.Attachment, 0, end-from)
	for _, t := range tickets[from:end] {
		cards = append(cards, s.myTicketCard(t, user, from))
	}
	return myTicketsChunk{message: message, cards: cards}
}

// myTicketsHeader — заголовок сообщения со срезом заявок. Первое сообщение
// показывает общее количество, остальные — свой диапазон.
func myTicketsHeader(from, end, total int) string {
	if from == 0 {
		return fmt.Sprintf("**Ваши активные заявки** (%d):", total)
	}
	return fmt.Sprintf("Заявки %d–%d из %d:", from+1, end, total)
}

// myTicketCard собирает карточку заявки списка. Заголовок ведёт на заявку
// (title_link) — отдельной кнопки «Открыть заявку» нет, вместо неё у заявки
// владельца появляются действия по статусу. Веб-ссылка доступна только
// веб-пользователю (Keycloak) с coarse-правом ticket:read; остальным заголовок
// ведёт на deep-link плагина, который открывается внутри Mattermost.
func (s *MattermostService) myTicketCard(t *models.Ticket, user *models.UserData, from int) mattermost.Attachment {
	title := t.Title
	if t.TicketNumber != nil {
		title = fmt.Sprintf("№%d — %s", *t.TicketNumber, t.Title)
	}

	fields := []mattermost.AttachmentField{
		{Title: "Статус", Value: mmStatusLabels[t.Status], Short: true},
		{Title: "Создана", Value: t.CreatedAt.Format("02.01.2006"), Short: true},
	}
	if t.Site != nil && t.Site.Name != "" {
		fields = append(fields, mattermost.AttachmentField{Title: "Площадка", Value: t.Site.Name, Short: true})
	}

	titleLink := pluginDeepLink(t.ID)
	if s.showWebLink(user, t.RealmID) {
		titleLink = s.taskLink(t.ID)
	}

	card := mattermost.Attachment{
		Title:     title,
		TitleLink: titleLink,
		Text:      cardSummary(t.Description),
		Color:     mmStatusColors[t.Status],
		Fields:    fields,
		Buttons:   s.myTicketActionButtons(t, user.ID, from),
	}
	return card
}

// cardSummary — описание заявки для карточки в посте. Текст схлопывается в одну
// строку (карточка — это строчка в ленте, а не абзац) и подрезается, чтобы одна
// заявка с длинным описанием не растянула весь список.
const cardSummaryLimit = 200

func cardSummary(text string) string {
	summary := strings.Join(strings.Fields(text), " ")
	if len([]rune(summary)) <= cardSummaryLimit {
		return summary
	}
	return string([]rune(summary)[:cardSummaryLimit]) + "…"
}

// myTicketActionButtons — действия владельца заявки в списке «Мои заявки».
// Правила те же, что в плагине: отменить можно только новую заявку,
// подтвердить решение и вернуть в работу — только решённую. Заявки, где
// пользователь не владелец, остаются без кнопок.
func (s *MattermostService) myTicketActionButtons(t *models.Ticket, ownerID uuid.UUID, from int) []mattermost.InteractiveButton {
	if t.Owner == nil || t.Owner.ID != ownerID {
		return nil
	}

	btnCtx := func(action string, status models.TicketStatus) map[string]string {
		return map[string]string{
			"action":    action,
			"ticket_id": t.ID.String(),
			"status":    string(status),
			"from":      strconv.Itoa(from),
		}
	}

	switch t.Status {
	case models.StatusOpen:
		return []mattermost.InteractiveButton{{
			Text: "Отменить заявку", Style: "danger",
			URL: s.actionURL(), Context: btnCtx(actionCancel, models.StatusCancelled),
			ID: ticketActionID(t.ID, actionCancel),
		}}
	case models.StatusResolved:
		return []mattermost.InteractiveButton{
			{
				Text: "Подтвердить решение", Style: "primary",
				URL: s.actionURL(), Context: btnCtx(actionConfirm, models.StatusClosed),
				ID: ticketActionID(t.ID, actionConfirm),
			},
			{
				Text: "Вернуть в работу",
				URL:  s.actionURL(), Context: btnCtx(actionReopen, models.StatusInProgress),
				ID: ticketActionID(t.ID, actionReopen),
			},
		}
	default:
		return nil
	}
}

// ticketActionID — action_id кнопки заявки. Он должен быть уникален в посте
// (сервер ищет действие по id в attachments поста и берёт первое совпадение, так
// что общий id у карточек увёл бы нажатие на чужую заявку) и состоять только из
// [A-Za-z0-9] — роут /posts/{id}/actions/{action_id} дефисы и подчёркивания не
// принимает, поэтому uuid режем до hex-префикса, а из имени действия выкидываем
// всё лишнее. Привязка к заявке, а не позиция в срезе: id не «едет», когда
// карточка уезжает из списка.
func ticketActionID(ticketID uuid.UUID, action string) string {
	suffix := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		}
		return -1
	}, strings.TrimPrefix(action, "ticket_"))
	if suffix == "" {
		suffix = "act"
	}
	return "t" + strings.ReplaceAll(ticketID.String(), "-", "")[:12] + suffix
}

// taskLink — абсолютная ссылка на заявку в веб-приложении. Абсолютная нужна для
// мобильного клиента Mattermost: site-relative адрес здесь не откроется.
func (s *MattermostService) taskLink(ticketID uuid.UUID) string {
	if s.baseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/tasks/%s", s.baseURL, ticketID.String())
}

// showWebLink решает, показывать ли пользователю веб-ссылку на заявку.
// Веб-ссылка ведёт на /tasks/:id, который доступен только пользователям из
// Keycloak: у пользователей, созданных из Mattermost (Source=mattermost), нет
// учётной записи в вебе. Второе условие — coarse-право ticket:read в домене
// заявки: без него страница отдаст отказ. Ошибка проверки трактуется как отказ
// (fail-closed на ссылке, доставка сообщения при этом не страдает).
func (s *MattermostService) showWebLink(user *models.UserData, realmID *uuid.UUID) bool {
	if user == nil || user.Source.IsFromMattermost() {
		return false
	}
	if s.policies == nil || realmID == nil {
		return false
	}
	allowed, err := s.policies.Enforce(user.ID.String(), realmID.String(), string(access.ResourceTicket), string(access.Read))
	if err != nil {
		logger.Warn("failed to check ticket read policy for mattermost web link",
			logger.StringAttr("user_id", user.ID.String()),
			logger.ErrAttr(err),
		)
		return false
	}
	return allowed
}

// sendStatusMessage отвечает на команду «мои | статус | заявки» списком активных
// заявок карточками — тем же содержимым, что и кнопка «Мои заявки», но обычным
// сообщением в канале. Длинный список уходит несколькими сообщениями.
func (s *MattermostService) sendStatusMessage(ctx context.Context, ch *mmChannel) error {
	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	tickets, err := s.myActiveTickets(ctx, ch.Settings.RealmID, user)
	if err != nil {
		return err
	}

	return s.sendMyTicketsChunks(ctx, ch.Settings.BotToken, ch.ChannelID, s.myTicketsChunks(tickets, user))
}

// sendMyTicketsChunks отправляет список заявок одним или несколькими сообщениями
// с паузой между ними. Ошибка одного сообщения не отменяет остальные: список
// на 27 заявок не должен пропадать из-за одного отказа.
func (s *MattermostService) sendMyTicketsChunks(ctx context.Context, botToken, channelID string, chunks []myTicketsChunk) error {
	var firstErr error
	for i, chunk := range chunks {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(myTicketsPostDelay):
			}
		}
		if _, err := s.most.Post.Create(botToken, mattermost.CreatePostDTO{
			ChannelID:   channelID,
			Message:     chunk.message,
			Attachments: chunk.cards,
		}); err != nil {
			logger.Warn("failed to send my tickets message",
				logger.StringAttr("channel_id", channelID),
				logger.IntAttr("part", i+1),
				logger.IntAttr("parts", len(chunks)),
				logger.ErrAttr(err),
			)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to send status message: %w", err)
			}
		}
	}
	return firstErr
}
