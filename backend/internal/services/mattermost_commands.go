package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
)

// cardDetailLimit — лимит описания в карточке заявки. Выше cardSummary (200) для
// одной заявки: карточка одна, растянуть ленту она не может, а обрезанное
// описание бесполезно — за ним идут за полным текстом в веб-приложении.
const cardDetailLimit = 1000

var mmPriorityLabels = map[models.Priority]string{
	models.PriorityLow:    "Низкий",
	models.PriorityMedium: "Средний",
	models.PriorityHigh:   "Высокий",
	models.PriorityUrgent: "Срочный",
}

// handleAttachFiles прикрепляет файлы Mattermost к заявке пользователя и, если
// передан commentText (текст сообщения, а не только номер), оставляет его
// комментарием к той же заявке. Если указан номер заявки — файлы ищутся по нему,
// и файлы прикрепляются, только если пользователь её создатель (иначе файлы не
// прикрепляются; комментарий всё же создаётся при наличии work-доступа). Без
// номера файлы прикрепляются к последней созданной пользователем заявке из
// recentTickets, если с момента создания прошло не более 30 минут.
func (s *MattermostService) handleAttachFiles(ctx context.Context, ch *mmChannel, ticketNumber int, fileIDs []string, commentText string) error {
	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	var ticketID uuid.UUID
	numberFromRecent := 0

	if ticketNumber > 0 {
		mode := readableMode
		tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
			Number:  &ticketNumber,
			RealmID: &ch.Settings.RealmID,
			Actor:   &models.Actor{ID: user.ID},
			Mode:    &mode,
		})
		if err != nil || len(tickets) == 0 {
			s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", ticketNumber))
			return nil
		}
		ticket := tickets[0]
		ticketID = ticket.ID
		if ticket.Creator.ID != user.ID {
			if commentText == "" {
				s.sendDMBestEffort(ch, "Прикреплять файлы может только создатель заявки")
				return nil
			}
			return s.commentAndReply(ctx, ch, ticketID, commentText)
		}
		numberFromRecent = ticketNumber
	} else {
		val, ok := s.recentTickets.Load(ch.MmUserID + ":" + ch.ChannelID)
		if !ok {
			s.sendDMBestEffort(ch, "Не найдена заявка для прикрепления файлов. Отправьте номер заявки (например, №123) вместе с файлами")
			return nil
		}
		rt := val.(*recentTicket)
		if time.Since(rt.createdAt) > 30*time.Minute {
			s.sendDMBestEffort(ch, "Прошло более 30 минут с создания заявки. Укажите номер заявки (например, №123) вместе с файлами")
			return nil
		}
		ticketID = rt.id
	}

	// В случае «файлы + текст» комментарий и файлы создаются атомарно и связываются
	// (файлы привязываются к комментарию через comment_id). Без текста файлы просто
	// прикрепляются к заявке.
	fileDTOs := make([]*models.UploadAttachmentDTO, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		data, err := s.most.Client.DownloadFile(ch.Settings.BotToken, fileID)
		if err != nil {
			logger.Warn("failed to download MM file for attach", logger.StringAttr("file_id", fileID), logger.ErrAttr(err))
			continue
		}
		info, err := s.most.Client.GetFileInfo(ch.Settings.BotToken, fileID)
		if err != nil {
			logger.Warn("failed to get MM file info for attach", logger.StringAttr("file_id", fileID), logger.ErrAttr(err))
			continue
		}
		fileName := info.Name
		if fileName == "" {
			fileName = fileID
		}
		fileDTOs = append(fileDTOs, &models.UploadAttachmentDTO{
			EntityType: "ticket",
			EntityID:   ticketID,
			FileName:   fileName,
			FileSize:   info.Size,
			MimeType:   info.MimeType,
			File:       bytes.NewReader(data),
			UploadedBy: user.ID,
			Realm:      ch.Settings.RealmID.String(),
		})
	}

	attached := 0
	commentCreated := false

	if commentText != "" {
		_, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
			Text:       commentText,
			TicketID:   ticketID,
			IsInternal: false,
			Type:       "",
			UserID:     user.ID,
			Realm:      ch.Settings.RealmID.String(),
			Files:      fileDTOs,
		})
		if err != nil {
			if errors.Is(err, models.ErrPermissionDenied) {
				s.sendDMBestEffort(ch, "Нет прав на комментарий к этой заявке")
				return nil
			}
			logger.Warn("failed to create comment with files", logger.StringAttr("ticket_id", ticketID.String()), logger.ErrAttr(err))
			s.sendDMBestEffort(ch, "Не удалось сохранить текст комментария")
			return nil
		}
		attached = len(fileDTOs)
		commentCreated = true
	} else {
		for _, fileDTO := range fileDTOs {
			att, err := s.tickets.UploadAttachment(ctx, nil, fileDTO)
			if err != nil {
				logger.Warn("failed to upload file as attachment", logger.StringAttr("ticket_id", ticketID.String()), logger.ErrAttr(err))
				continue
			}
			logger.Info("file attached to ticket",
				logger.StringAttr("ticket_id", ticketID.String()),
				logger.StringAttr("attachment_id", att.ID.String()),
				logger.StringAttr("file_name", fileDTO.FileName),
			)
			attached++
		}
	}

	var reply string
	if numberFromRecent > 0 {
		reply = fmt.Sprintf("К заявке №%d прикреплено файлов: %d", numberFromRecent, attached)
	} else {
		reply = fmt.Sprintf("К заявке прикреплено файлов: %d", attached)
	}
	if commentCreated {
		reply += "\nКомментарий добавлен"
	}
	if attached == 0 && len(fileDTOs) > 0 {
		reply = "Файлы не удалось прикрепить к заявке"
	}
	s.sendDMBestEffort(ch, reply)
	return nil
}

// commentAndReply создаёт комментарий к тикету (work-доступ) и отправляет
// пользователю DM с результатом. Используется, когда файлы не прикрепить
// (пользователь не создатель), но комментарий оставить можно.
func (s *MattermostService) commentAndReply(ctx context.Context, ch *mmChannel, ticketID uuid.UUID, commentText string) error {
	if commentText == "" {
		return nil
	}
	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}
	if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:       commentText,
		TicketID:   ticketID,
		IsInternal: false,
		Type:       "",
		UserID:     user.ID,
		Realm:      ch.Settings.RealmID.String(),
	}); err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ch, "Нет прав комментировать эту заявку")
			return nil
		}
		return fmt.Errorf("failed to create comment: %w", err)
	}
	s.sendDMBestEffort(ch, "Комментарий добавлен. Файлы может прикреплять только создатель заявки")
	return nil
}

// handleTextWithFiles обрабатывает сообщение с файлами и текстом: прикрепляет
// файлы и оставляет текст комментарием к той же заявке. Номер заявки берётся из
// сообщения (если есть, например «№123 ...»), иначе файлы и комментарий идут к
// последней созданной пользователем заявке из recentTickets.
func (s *MattermostService) handleTextWithFiles(ctx context.Context, ch *mmChannel, msg string, fileIDs []string) error {
	number := 0
	text := ""
	if m := commentCommands.FindStringSubmatch(msg); m != nil {
		number, _ = strconv.Atoi(m[1])
		text = strings.TrimSpace(m[2])
	} else {
		text = strings.TrimSpace(msg)
	}
	return s.handleAttachFiles(ctx, ch, number, fileIDs, text)
}

// handleTicketCard отвечает на голый номер заявки («№123») карточкой: статус,
// приоритет, участники, срок и действия владельца.
//
// Право на просмотр здесь обязательно и обеспечивается двумя шагами. Поиск по
// номеру идёт через Tickets.Get в режиме readable (автор/заказчик/исполнитель/
// группа тикета), поэтому чужая заявка в него не попадает; найденный ID всё
// равно досматривается через ticketDetail → Tickets.GetByID — он гоняет
// атрибутную модель доступа и вернёт ErrPermissionDenied на чужую заявку.
// Отказ показываем тем же «не найдена», что и для несуществующего номера:
// сообщать о существовании чужой заявки нельзя.
func (s *MattermostService) handleTicketCard(ctx context.Context, ch *mmChannel, message string) error {
	parts := attachCommands.FindStringSubmatch(message)
	number, _ := strconv.Atoi(parts[1])

	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	mode := readableMode
	found, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		Number:  &number,
		RealmID: &ch.Settings.RealmID,
		Actor:   &models.Actor{ID: user.ID, Name: user.Username},
		Mode:    &mode,
	})
	if err != nil {
		return fmt.Errorf("failed to find ticket by number: %w", err)
	}
	if len(found) == 0 {
		s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", number))
		return nil
	}

	detail, err := s.ticketDetail(ctx, ch.Settings, ch.MmUserID, found[0].ID)
	if err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", number))
			return nil
		}
		return fmt.Errorf("failed to get ticket detail: %w", err)
	}

	if _, err := s.most.Post.Create(ch.Settings.BotToken, mattermost.CreatePostDTO{
		ChannelID:   ch.ChannelID,
		Message:     fmt.Sprintf("Заявка №%d", detail.Number),
		Attachments: []mattermost.Attachment{s.ticketCard(detail)},
	}); err != nil {
		return fmt.Errorf("failed to send ticket card: %w", err)
	}
	return nil
}

// ticketCard собирает карточку заявки для ответа на «№N». Заголовок ведёт на
// заявку через title_link: отдельного поля url у вложения нет, клиенты рисуют
// ссылкой только заголовок. Кнопки — те же, что у карточки в списке «Мои
// заявки», но без post_id: applyListEdit правку списка не делает, а по
// standalone-карточке её и нечего править, поэтому действие уходит в диалог
// подтверждения и меняет статус через общий TicketService.Update.
func (s *MattermostService) ticketCard(detail *models.PluginTicketDetail) mattermost.Attachment {
	title := detail.Title
	fields := []mattermost.AttachmentField{
		{Title: "Статус", Value: mmStatusLabels[detail.Status], Short: true},
		{Title: "Приоритет", Value: mmPriorityLabels[detail.Priority], Short: true},
	}
	if detail.Category != nil && detail.Category.Name != "" {
		fields = append(fields, mattermost.AttachmentField{Title: "Категория", Value: detail.Category.Name, Short: true})
	}
	if detail.Site != nil && detail.Site.Name != "" {
		fields = append(fields, mattermost.AttachmentField{Title: "Площадка", Value: detail.Site.Name, Short: true})
	}
	fields = append(fields, mattermost.AttachmentField{
		Title: "Создана", Value: detail.CreatedAt.Format("02.01.2006 15:04"), Short: true,
	})
	if detail.DueDate != nil {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Срок", Value: detail.DueDate.Format("02.01.2006"), Short: true,
		})
	}
	if detail.Assignee != nil && detail.Assignee.Username != "" {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Исполнитель", Value: detail.Assignee.Username, Short: true,
		})
	}
	if detail.Owner != nil && detail.Owner.Username != "" {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Заказчик", Value: detail.Owner.Username, Short: true,
		})
	}
	if len(detail.Attachments) > 0 {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Вложения", Value: strconv.Itoa(len(detail.Attachments)), Short: true,
		})
	}

	card := mattermost.Attachment{
		Title:   title,
		Text:    detailText(detail.Description),
		Color:   mmStatusColors[detail.Status],
		Fields:  fields,
		Buttons: s.detailActionButtons(detail),
	}
	if detail.Link != "" {
		card.TitleLink = detail.Link
	}
	return card
}

// detailActionButtons — действия владельца на карточке заявки. Правила те же, что
// в плагине: отмена только для новой заявки, подтверждение решения и возврат в
// работу — только для решённой. Заявка не владельца остаётся без кнопок.
func (s *MattermostService) detailActionButtons(detail *models.PluginTicketDetail) []mattermost.InteractiveButton {
	if detail.Owner == nil {
		return nil
	}
	btnCtx := func(action string, status models.TicketStatus) map[string]string {
		return map[string]string{
			"action":    action,
			"ticket_id": detail.ID.String(),
			"status":    string(status),
		}
	}

	switch detail.Status {
	case models.StatusOpen:
		if !detail.CanCancel {
			return nil
		}
		return []mattermost.InteractiveButton{{
			Text: "Отменить заявку", Style: "danger",
			URL: s.actionURL(), Context: btnCtx(actionCancel, models.StatusCancelled),
			ID: ticketActionID(detail.ID, actionCancel),
		}}
	case models.StatusResolved:
		var buttons []mattermost.InteractiveButton
		if detail.CanConfirm {
			buttons = append(buttons, mattermost.InteractiveButton{
				Text: "Подтвердить решение", Style: "primary",
				URL: s.actionURL(), Context: btnCtx(actionConfirm, models.StatusClosed),
				ID: ticketActionID(detail.ID, actionConfirm),
			})
		}
		if detail.CanReopen {
			buttons = append(buttons, mattermost.InteractiveButton{
				Text: "Вернуть в работу",
				URL:  s.actionURL(), Context: btnCtx(actionReopen, models.StatusInProgress),
				ID: ticketActionID(detail.ID, actionReopen),
			})
		}
		return buttons
	default:
		return nil
	}
}

// detailText — описание заявки для карточки: переносы схлопнуты в абзацы, а не
// в одну строку (здесь, в отличие от списка, описание — содержание карточки),
// и подрезано до cardDetailLimit.
func detailText(text string) string {
	paragraphs := strings.Split(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\n")
	kept := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	out := strings.Join(kept, "\n")
	if len([]rune(out)) <= cardDetailLimit {
		return out
	}
	return string([]rune(out)[:cardDetailLimit]) + "…"
}

// handleComment создаёт комментарий к заявке из личного сообщения Mattermost.
// Синтаксис: «№123 текст комментария». Пользователь резолвится по mattermost_id,
// комментарий проходит ту же проверку work-доступа, что и написанный в программе.
func (s *MattermostService) handleComment(ctx context.Context, ch *mmChannel, message string) error {
	parts := commentCommands.FindStringSubmatch(message)
	number, _ := strconv.Atoi(parts[1])
	text := strings.TrimSpace(parts[2])

	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	mode := readableMode
	tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		Number:  &number,
		RealmID: &ch.Settings.RealmID,
		Actor:   &models.Actor{ID: user.ID},
		Mode:    &mode,
	})
	if err != nil || len(tickets) == 0 {
		s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", number))
		return nil
	}
	ticket := tickets[0]

	if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:       text,
		TicketID:   ticket.ID,
		IsInternal: false,
		Type:       "",
		UserID:     user.ID,
		Realm:      ch.Settings.RealmID.String(),
	}); err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ch, fmt.Sprintf("Нет прав комментировать заявку №%d", number))
			return nil
		}
		return fmt.Errorf("failed to create comment from mattermost: %w", err)
	}

	s.sendDMBestEffort(ch, fmt.Sprintf("Комментарий добавлен к заявке №%d", number))
	return nil
}
