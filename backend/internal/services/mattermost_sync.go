package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

// resolveOrCreateUser находит или создаёт системного пользователя по userID
// из Mattermost, возвращая его данные (ID, имя и привязанную площадку). Сначала
// ищет уже сохранённую связку; если её нет — пытается сопоставить с существующим
// пользователем (по email, username или ФИО) и при неудаче создаёт нового.
// Реализовано так, чтобы внешний Mattermost-пользователь всегда мог создавать
// заявки без ручной регистрации в системе.
func (s *MattermostService) resolveOrCreateUser(ctx context.Context, realmID uuid.UUID, mmUserID string, siteID *uuid.UUID) (*models.UserData, error) {
	existing, err := s.users.GetByMattermostID(ctx, mmUserID)
	if err == nil {
		s.ensureKnownUserRealm(ctx, realmID, existing.ID, siteID, existing.ID, existing.Username)
		return existing, nil
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get realm settings: %w", err)
	}

	mmUser, err := s.most.Client.GetUser(settings.BotToken, mmUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mattermost user: %w", err)
	}

	sysUsers, err := s.users.GetAll(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get system users: %w", err)
	}

	if matched, userID, username := matchByEmail(sysUsers, mmUser.Email); matched {
		s.ensureLinkAndRealm(ctx, realmID, userID, mmUser.Id, siteID, userID, username)
		return &models.UserData{ID: userID, Username: username, SiteID: userSiteByID(sysUsers, userID), Source: userSourceByID(sysUsers, userID)}, nil
	}

	if matched, userID, username := matchByUsername(sysUsers, mmUser.Username); matched {
		s.ensureLinkAndRealm(ctx, realmID, userID, mmUser.Id, siteID, userID, username)
		return &models.UserData{ID: userID, Username: username, SiteID: userSiteByID(sysUsers, userID), Source: userSourceByID(sysUsers, userID)}, nil
	}

	mmFio := buildFIO(mmUser.FirstName, cleanMMLastName(mmUser.LastName))
	if mmFio != "" {
		for _, sysU := range sysUsers {
			if buildFIO(sysU.FirstName, sysU.LastName) == mmFio {
				s.ensureLinkAndRealm(ctx, realmID, sysU.ID, mmUser.Id, siteID, sysU.ID, sysU.Username)
				return sysU, nil
			}
		}
	}

	newUserID := uuid.New()
	mattermostID := mmUser.Id
	userDTO := &models.UserDataDTO{
		ID:           newUserID,
		MattermostID: &mattermostID,
		SiteID:       siteID,
		Username:     mmUser.Username,
		FirstName:    mmUser.FirstName,
		LastName:     cleanMMLastName(mmUser.LastName),
		Email:        mmUser.Email,
		IsActive:     true,
		Source:       models.UserSourceMattermost,
	}
	if err := s.users.CreateSeveral(ctx, nil, []*models.UserDataDTO{userDTO}); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	if err := s.ensureRealmMembership(ctx, newUserID, realmID, newUserID, mmUser.Username); err != nil {
		logger.Warn("failed to add created user to realm",
			logger.StringAttr("user_id", newUserID.String()),
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
	}

	logger.Info("created user from mattermost",
		logger.StringAttr("mm_user_id", mmUserID),
		logger.StringAttr("user_id", newUserID.String()),
	)

	newUser := &models.UserData{
		ID:       newUserID,
		Username: mmUser.Username,
		Source:   models.UserSourceMattermost,
	}
	if siteID != nil {
		value := siteID.String()
		newUser.SiteID = &value
	}

	return newUser, nil
}

// userSiteByID возвращает привязанную площадку пользователя из списка (системные
// пользователи уже содержат SiteID после чтения из БД).
func userSiteByID(users []*models.UserData, id uuid.UUID) *string {
	for _, u := range users {
		if u.ID == id {
			return u.SiteID
		}
	}
	return nil
}

// userSourceByID возвращает источник пользователя из списка. Пустое значение и
// отсутствие записи трактуются как keycloak — так же нормализует источник
// postgres.userRepo (mapUserSource), поэтому веб-ссылка таким пользователям
// разрешена (с поправкой на coarse-право).
func userSourceByID(users []*models.UserData, id uuid.UUID) models.UserSource {
	for _, u := range users {
		if u.ID == id {
			if u.Source == "" {
				return models.UserSourceKeycloak
			}
			return u.Source
		}
	}
	return models.UserSourceKeycloak
}

// ensureRealmMembership добавляет пользователя в realm с ролью «user», если он
// там ещё не состоит, и публикует событие политики (перезагрузка Casbin + аудит),
// чтобы новая привязка user→realm→role стала доступна для проверки прав сразу же.
// Возвращает ошибку, если роль «user» не найдена или вставка не удалась; если
// пользователь уже в realm — успех без события.
func (s *MattermostService) ensureRealmMembership(ctx context.Context, userID uuid.UUID, realmID uuid.UUID, actorID uuid.UUID, actorName string) error {
	ur, urErr := s.userRealms.GetByUserAndRealm(ctx, userID, realmID)
	if urErr == nil && ur != nil {
		return nil
	}

	roleID, err := s.roles.GetIDBySlug(ctx, realmID, "user")
	if err != nil {
		return fmt.Errorf("failed to get default role: %w", err)
	}

	if err := s.userRealms.CreateSeveral(ctx, nil, []*models.UserRealmDTO{
		{UserID: userID, RealmID: realmID, RoleID: &roleID, IsActive: true},
	}); err != nil {
		return fmt.Errorf("failed to add user to realm: %w", err)
	}

	event := events.PolicyEvent{
		ChangedBy:     actorID,
		ChangedByName: actorName,
		Action:        "add_user_realm",
		EntityType:    "users",
		EntityID:      &userID,
		RealmID:       &realmID,
	}
	if err := event.SetNewValues(map[string]any{
		"realmId": realmID,
		"role":    "user",
	}); err != nil {
		bestEffortError("failed to set audit new values after adding user to realm", err,
			map[string]string{"user_id": userID.String(), "realm_id": realmID.String()})
	}
	s.eventBus.Notify(event)

	return nil
}

// ensureKnownUserRealm готовит уже сопоставленного по mattermost_id пользователя:
// при необходимости проставляет площадку и гарантирует членство в realm.
// В отличие от ensureLinkAndRealm не трогает mattermost_id — он уже совпадает по
// определению fast-path, поэтому запись в БД давала бы только обновление
// updated_at на каждый вызов (например, /plugin/context при каждом переключении
// канала), создавая мёртвые кортежи без изменения данных.
func (s *MattermostService) ensureKnownUserRealm(ctx context.Context, realmID uuid.UUID, userID uuid.UUID, siteID *uuid.UUID, actorID uuid.UUID, actorName string) {
	if siteID != nil {
		if err := s.users.UpdateMMAndSite(ctx, nil, &models.UserDataDTO{
			ID:     userID,
			SiteID: siteID,
		}); err != nil {
			logger.Warn("failed to update user site", logger.ErrAttr(err))
		}
	}
	if err := s.ensureRealmMembership(ctx, userID, realmID, actorID, actorName); err != nil {
		logger.Warn("failed to add user to realm", logger.ErrAttr(err))
	}
}

// ensureLinkAndRealm привязывает системного пользователя к его Mattermost
// userID (через users.mattermost_id), добавляет его в realm с ролью «user»,
// если он там ещё не состоит, и обновляет site_id, если он передан. Ошибки
// некритичны (только логируются), чтобы не прерывать основной поток
// сопоставления пользователя.
func (s *MattermostService) ensureLinkAndRealm(ctx context.Context, realmID uuid.UUID, userID uuid.UUID, mmUserID string, siteID *uuid.UUID, actorID uuid.UUID, actorName string) {
	mmCopy := mmUserID
	if err := s.users.UpdateMMAndSite(ctx, nil, &models.UserDataDTO{
		ID:           userID,
		MattermostID: &mmCopy,
		SiteID:       siteID,
	}); err != nil {
		logger.Warn("failed to update mattermost id/site", logger.ErrAttr(err))
	}
	if err := s.ensureRealmMembership(ctx, userID, realmID, actorID, actorName); err != nil {
		logger.Warn("failed to add user to realm", logger.ErrAttr(err))
	}
}

// matchByEmail ищет системного пользователя по email (без учёта регистра).
func matchByEmail(sysUsers []*models.UserData, email string) (bool, uuid.UUID, string) {
	if email == "" {
		return false, uuid.Nil, ""
	}
	for _, u := range sysUsers {
		if u.Email != "" && strings.EqualFold(u.Email, email) {
			return true, u.ID, u.Username
		}
	}
	return false, uuid.Nil, ""
}

// matchByUsername ищет системного пользователя по username (без учёта регистра).
func matchByUsername(sysUsers []*models.UserData, username string) (bool, uuid.UUID, string) {
	if username == "" {
		return false, uuid.Nil, ""
	}
	for _, u := range sysUsers {
		if u.Username != "" && strings.EqualFold(u.Username, username) {
			return true, u.ID, u.Username
		}
	}
	return false, uuid.Nil, ""
}

// buildFIO приводит «Имя Фамилия» к единому нижнему регистру и убирает лишние
// пробелы — чтобы сравнивать записи о людях из разных источников независимо
// от регистра написания.
func buildFIO(firstName, lastName string) string {
	return strings.ToLower(strings.TrimSpace(firstName + " " + lastName))
}

// lastNamePhoneLabelRe — хвостовая надпись перед телефоном, которую добавляют к фамилии:
// «тел.», «телефон», «моб.», «phone», «ext».
var lastNamePhoneLabelRe = regexp.MustCompile(`(?i)\s*(?:тел|телефон|моб|phone|ext)\s*\.?\s*$`)

// cleanMMLastName очищает фамилию Mattermost от «хвостов» с цифрами, которые пользователи
// дописывают в last_name: телефон («Иванов +7 999 123-45-67», «Иванов, тел. 89991234567»)
// или внутренние номера в скобках («Хасанова (603, 604)»).
// Отрезает всё от первого символа, который цифра или «+», затем снимает хвостовые
// разделители и надписи (тел/телефон/моб/phone/ext). Если после обрезки ничего не
// осталось (вся фамилия — телефон), возвращает исходное значение, чтобы не создавать
// пользователя без фамилии. В отличие от Keycloak-синка (extractInternalNumber) номера
// из скобок не сохраняются: для пользователей из Mattermost внутренний номер не заводится.
func cleanMMLastName(lastName string) string {
	trimmed := strings.TrimSpace(lastName)
	cut := strings.IndexFunc(trimmed, func(r rune) bool {
		return r == '+' || unicode.IsDigit(r)
	})
	if cut < 0 {
		return trimmed
	}

	head := trimmed[:cut]
	for {
		head = lastNamePhoneLabelRe.ReplaceAllString(head, "")
		head = strings.TrimSpace(strings.Trim(head, " \t,;:-–—()[]"))
		head = strings.Join(strings.Fields(head), " ")
		if lastNamePhoneLabelRe.MatchString(head) {
			continue
		}
		break
	}
	if head == "" {
		return trimmed
	}
	return head
}

// handleSync — обработчик команды «синхронизировать [команды]»: запускает
// SyncRealmUsers и отчитывается результатом в DM. Доступна только администратору
// realm; необязательными аргументами можно ограничить синк конкретными командами
// Mattermost. Логика самого сопоставления пользователей живёт в SyncRealmUsers,
// чтобы её же мог вызвать веб (кнопка в настройках realm).
func (s *MattermostService) handleSync(ctx context.Context, ch *mmChannel, message string) error {
	sender, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve sender: %w", err)
	}

	if !s.isRealmSupervisor(ctx, sender.ID, ch.Settings.RealmID) {
		if err := s.most.DM.Send(ch.Settings.BotToken, ch.Settings.BotUserID, ch.MmUserID,
			"Только администраторы могут синхронизировать пользователей"); err != nil {
			return fmt.Errorf("failed to send no-permission message: %w", err)
		}
		return nil
	}

	teamNames := []string{}
	if parts := syncCommands.FindStringSubmatch(message); len(parts) > 1 && parts[1] != "" {
		for _, t := range strings.Split(parts[1], ",") {
			if t = strings.TrimSpace(t); t != "" {
				teamNames = append(teamNames, t)
			}
		}
	}

	result, err := s.SyncRealmUsers(ctx, ch.Settings.RealmID, &models.Actor{ID: sender.ID, Name: sender.Username}, teamNames)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Синхронизация завершена. Создано: %d, привязано: %d, с ошибкой: %d", result.Created, result.Linked, result.Failed)
	if err := s.most.DM.Send(ch.Settings.BotToken, ch.Settings.BotUserID, ch.MmUserID, msg); err != nil {
		bestEffortError("failed to send sync result", err, map[string]string{"mm_user_id": ch.MmUserID})
	}
	return nil
}

// SyncUsersResult — итог импорта пользователей Mattermost в realm.
type SyncUsersResult struct {
	Created int `json:"created"`
	Linked  int `json:"linked"`
	Failed  int `json:"failed"`
}

// SyncRealmUsers сопоставляет пользователей Mattermost с системными (по
// mattermost_id, затем по email, username и ФИО) и создаёт недостающих
// локальных пользователей в указанном realm. actor идёт в аудит привязок
// пользователя к realm. Пустой teamNames означает «все пользователи сервера».
// Запускается из бота (команда «синхронизировать») и из веба (кнопка в
// настройках realm), поэтому синхронизация разрешена только администратору realm.
func (s *MattermostService) SyncRealmUsers(ctx context.Context, realmID uuid.UUID, actor *models.Actor, teamNames []string) (*SyncUsersResult, error) {
	if actor == nil || !s.isRealmSupervisor(ctx, actor.ID, realmID) {
		return nil, models.ErrPermissionDenied
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mattermost settings: %w", err)
	}
	if !settings.IsActive || settings.BotToken == "" {
		return nil, fmt.Errorf("%w: интеграция Mattermost не настроена", models.ErrInvalidInput)
	}

	mmUsers, err := s.fetchMMUsers(settings.BotToken, teamNames)
	if err != nil {
		return nil, err
	}

	sysUsers, err := s.users.GetAll(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get system users: %w", err)
	}

	sysByEmail := make(map[string]*models.UserData, len(sysUsers))
	sysByUsername := make(map[string]*models.UserData, len(sysUsers))
	sysByFIO := make(map[string]*models.UserData, len(sysUsers))
	for _, u := range sysUsers {
		if u.Email != "" {
			sysByEmail[strings.ToLower(u.Email)] = u
		}
		if u.Username != "" {
			sysByUsername[strings.ToLower(u.Username)] = u
		}
		if key := buildFIO(u.FirstName, u.LastName); key != "" {
			sysByFIO[key] = u
		}
	}

	result := &SyncUsersResult{}
	seen := make(map[string]struct{}, len(mmUsers))

	for _, mmU := range mmUsers {
		if _, ok := seen[mmU.Id]; ok {
			continue
		}
		seen[mmU.Id] = struct{}{}

		existing, existingErr := s.users.GetByMattermostID(ctx, mmU.Id)
		if existingErr == nil {
			if err := s.ensureRealmMembership(ctx, existing.ID, realmID, actor.ID, actor.Name); err != nil {
				logger.Warn("failed to add user to realm",
					logger.StringAttr("mm_user_id", mmU.Id),
					logger.ErrAttr(err),
				)
				result.Failed++
			} else {
				result.Linked++
			}
			continue
		}

		matched := false

		if mmU.Email != "" {
			if sysU, ok := sysByEmail[strings.ToLower(mmU.Email)]; ok {
				s.ensureLinkAndRealm(ctx, realmID, sysU.ID, mmU.Id, nil, actor.ID, actor.Name)
				result.Linked++
				matched = true
			}
		}

		if !matched && mmU.Username != "" {
			if sysU, ok := sysByUsername[strings.ToLower(mmU.Username)]; ok {
				s.ensureLinkAndRealm(ctx, realmID, sysU.ID, mmU.Id, nil, actor.ID, actor.Name)
				result.Linked++
				matched = true
			}
		}

		if !matched {
			if fio := buildFIO(mmU.FirstName, cleanMMLastName(mmU.LastName)); fio != "" {
				if sysU, ok := sysByFIO[fio]; ok {
					s.ensureLinkAndRealm(ctx, realmID, sysU.ID, mmU.Id, nil, actor.ID, actor.Name)
					result.Linked++
					matched = true
				}
			}
		}

		if matched {
			continue
		}

		newUserID := uuid.New()
		mattermostID := mmU.Id
		userDTO := &models.UserDataDTO{
			ID:           newUserID,
			MattermostID: &mattermostID,
			Username:     mmU.Username,
			FirstName:    mmU.FirstName,
			LastName:     cleanMMLastName(mmU.LastName),
			Email:        mmU.Email,
			IsActive:     true,
			Source:       models.UserSourceMattermost,
		}
		if err := s.users.CreateSeveral(ctx, nil, []*models.UserDataDTO{userDTO}); err != nil {
			logger.Warn("failed to create user from mattermost",
				logger.StringAttr("mm_user_id", mmU.Id),
				logger.ErrAttr(err),
			)
			result.Failed++
			continue
		}
		if err := s.ensureRealmMembership(ctx, newUserID, realmID, actor.ID, actor.Name); err != nil {
			logger.Warn("failed to add user to realm",
				logger.StringAttr("mm_user_id", mmU.Id),
				logger.ErrAttr(err),
			)
			// Пользователь создан, но в realm не добавлен: для вызывающего это
			// неудача, поэтому в Created он не попадает — счётчики взаимоисключающие.
			result.Failed++
			continue
		}
		result.Created++
	}

	logger.Info("mattermost users synced",
		logger.StringAttr("realm_id", realmID.String()),
		logger.Int32Attr("created", int32(result.Created)),
		logger.Int32Attr("linked", int32(result.Linked)),
		logger.Int32Attr("failed", int32(result.Failed)),
	)

	return result, nil
}

// fetchMMUsers получает список пользователей Mattermost для синка. Если
// переданы названия команд — берутся их участники (по одной команде за раз,
// ошибки по отдельным командам не прерывают остальные); иначе — все пользователи.
func (s *MattermostService) fetchMMUsers(botToken string, teamNames []string) ([]*model.User, error) {
	var mmUsers []*model.User

	if len(teamNames) > 0 {
		for _, teamName := range teamNames {
			team, err := s.most.Client.GetTeamByName(botToken, teamName)
			if err != nil {
				logger.Warn("failed to get team by name",
					logger.StringAttr("team", teamName),
					logger.ErrAttr(err),
				)
				continue
			}
			users, err := s.most.Client.GetUsersInTeam(botToken, team.Id)
			if err != nil {
				logger.Warn("failed to get users in team",
					logger.StringAttr("team", teamName),
					logger.ErrAttr(err),
				)
				continue
			}
			mmUsers = append(mmUsers, users...)
		}
	} else {
		var err error
		mmUsers, err = s.most.Client.GetAllUsers(botToken)
		if err != nil {
			return nil, fmt.Errorf("failed to get mattermost users: %w", err)
		}
	}

	return mmUsers, nil
}
