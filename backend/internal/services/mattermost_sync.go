package services

import (
	"context"
	"errors"
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
		s.ensureKnownUserRealm(ctx, &models.RealmMembershipDTO{
			RealmID:   realmID,
			UserID:    existing.ID,
			ActorID:   existing.ID,
			ActorName: existing.Username,
		}, siteID)
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
		s.ensureLinkAndRealm(ctx, &models.RealmMembershipDTO{
			RealmID:   realmID,
			UserID:    userID,
			ActorID:   userID,
			ActorName: username,
		}, mmUser.Id, siteID)
		return &models.UserData{ID: userID, Username: username, SiteID: userSiteByID(sysUsers, userID), Source: userSourceByID(sysUsers, userID)}, nil
	}

	if matched, userID, username := matchByUsername(sysUsers, mmUser.Username); matched {
		s.ensureLinkAndRealm(ctx, &models.RealmMembershipDTO{
			RealmID:   realmID,
			UserID:    userID,
			ActorID:   userID,
			ActorName: username,
		}, mmUser.Id, siteID)
		return &models.UserData{ID: userID, Username: username, SiteID: userSiteByID(sysUsers, userID), Source: userSourceByID(sysUsers, userID)}, nil
	}

	mmFio := buildFIO(mmUser.FirstName, cleanMMLastName(mmUser.LastName))
	if mmFio != "" {
		for _, sysU := range sysUsers {
			if buildFIO(sysU.FirstName, sysU.LastName) == mmFio {
				s.ensureLinkAndRealm(ctx, &models.RealmMembershipDTO{
					RealmID:   realmID,
					UserID:    sysU.ID,
					ActorID:   sysU.ID,
					ActorName: sysU.Username,
				}, mmUser.Id, siteID)
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
	if err := s.ensureRealmMembership(ctx, &models.RealmMembershipDTO{
		RealmID:   realmID,
		UserID:    newUserID,
		ActorID:   newUserID,
		ActorName: mmUser.Username,
	}); err != nil {
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
func (s *MattermostService) ensureRealmMembership(ctx context.Context, m *models.RealmMembershipDTO) error {
	ur, urErr := s.userRealms.GetByUserAndRealm(ctx, m.UserID, m.RealmID)
	if urErr == nil && ur != nil {
		return nil
	}

	roleID, err := s.roles.GetIDBySlug(ctx, m.RealmID, "user")
	if err != nil {
		return fmt.Errorf("failed to get default role: %w", err)
	}

	if err := s.userRealms.CreateSeveral(ctx, nil, []*models.UserRealmDTO{
		{UserID: m.UserID, RealmID: m.RealmID, RoleID: &roleID, IsActive: true},
	}); err != nil {
		return fmt.Errorf("failed to add user to realm: %w", err)
	}

	event := events.PolicyEvent{
		ChangedBy:     m.ActorID,
		ChangedByName: m.ActorName,
		Action:        "add_user_realm",
		EntityType:    "users",
		EntityID:      &m.UserID,
		RealmID:       &m.RealmID,
	}
	if err := event.SetNewValues(map[string]any{
		"realmId": m.RealmID,
		"role":    "user",
	}); err != nil {
		bestEffortError("failed to set audit new values after adding user to realm", err,
			map[string]string{"user_id": m.UserID.String(), "realm_id": m.RealmID.String()})
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
func (s *MattermostService) ensureKnownUserRealm(ctx context.Context, m *models.RealmMembershipDTO, siteID *uuid.UUID) {
	if siteID != nil {
		if err := s.users.UpdateMMAndSite(ctx, nil, &models.UserDataDTO{
			ID:     m.UserID,
			SiteID: siteID,
		}); err != nil {
			logger.Warn("failed to update user site", logger.ErrAttr(err))
		}
	}
	if err := s.ensureRealmMembership(ctx, m); err != nil {
		logger.Warn("failed to add user to realm", logger.ErrAttr(err))
	}
}

// ensureLinkAndRealm привязывает системного пользователя к его Mattermost
// userID (через users.mattermost_id), добавляет его в realm с ролью «user»,
// если он там ещё не состоит, и обновляет site_id, если он передан. Ошибки
// некритичны (только логируются), чтобы не прерывать основной поток
// сопоставления пользователя.
func (s *MattermostService) ensureLinkAndRealm(ctx context.Context, m *models.RealmMembershipDTO, mmUserID string, siteID *uuid.UUID) {
	mmCopy := mmUserID
	if err := s.users.UpdateMMAndSite(ctx, nil, &models.UserDataDTO{
		ID:           m.UserID,
		MattermostID: &mmCopy,
		SiteID:       siteID,
	}); err != nil {
		logger.Warn("failed to update mattermost id/site", logger.ErrAttr(err))
	}
	if err := s.ensureRealmMembership(ctx, m); err != nil {
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
	if result.Deactivated > 0 || result.Reactivated > 0 {
		msg += fmt.Sprintf(", деактивировано: %d, реактивировано: %d", result.Deactivated, result.Reactivated)
	}
	if err := s.most.DM.Send(ch.Settings.BotToken, ch.Settings.BotUserID, ch.MmUserID, msg); err != nil {
		bestEffortError("failed to send sync result", err, map[string]string{"mm_user_id": ch.MmUserID})
	}
	return nil
}

// SyncUsersResult — итог импорта пользователей Mattermost в realm.
// Deactivated/Reactivated — изменения активности локальных учётных записей,
// которые уже были связаны с Mattermost (не новые пользователи).
type SyncUsersResult struct {
	Created     int `json:"created"`
	Linked      int `json:"linked"`
	Failed      int `json:"failed"`
	Deactivated int `json:"deactivated"`
	Reactivated int `json:"reactivated"`
}

// setLocalUserActive обновляет флаг активности локальной учётной записи,
// связанной с Mattermost. Остальные поля DTO обязаны нести текущие значения:
// repo.UpdateAccount перезаписывает mattermost_id и internal_number безусловно.
// Realms не передаётся — привязки к realm здесь не меняются, поэтому защита
// кросс-реалмовых прав в UpdateAccount не срабатывает.
func (s *MattermostService) setLocalUserActive(ctx context.Context, user *models.UserData, active bool, actor *models.Actor) error {
	return s.users.UpdateAccount(ctx, &models.UpdateAccountDTO{
		ID:             user.ID,
		IsActive:       active,
		MattermostID:   user.MattermostID,
		InternalNumber: user.InternalNumber,
		Actor:          actor,
	})
}

// SyncRealmUsers — пользовательская точка входа (бот/веб): синхронизация
// разрешена только администратору realm. Пустой teamNames означает «все
// пользователи сервера».
func (s *MattermostService) SyncRealmUsers(ctx context.Context, realmID uuid.UUID, actor *models.Actor, teamNames []string) (*SyncUsersResult, error) {
	if actor == nil || !s.isRealmSupervisor(ctx, actor.ID, realmID) {
		return nil, models.ErrPermissionDenied
	}

	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	return s.syncRealmUsers(ctx, realmID, actor, teamNames)
}

// SyncAllActiveRealms — фоновая синхронизация всех активных интеграций.
// Запускается планировщиком, поэтому проверки прав администратора здесь нет:
// решение принимает сервер, а не пользовательский запрос. Ошибки отдельных
// realm не прерывают остальные и не валят задачу — только логируются.
func (s *MattermostService) SyncAllActiveRealms(ctx context.Context) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	settings, err := s.repo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active mattermost settings: %w", err)
	}

	actor := &models.Actor{ID: uuid.Nil, Name: "Mattermost sync"}
	failed := 0
	for _, st := range settings {
		if st.BotToken == "" {
			logger.Warn("mattermost auto sync skipped: empty bot token",
				logger.StringAttr("realm_id", st.RealmID.String()))
			failed++
			continue
		}
		result, err := s.syncRealmUsers(ctx, st.RealmID, actor, nil)
		if err != nil {
			logger.Error("mattermost auto sync failed",
				logger.StringAttr("realm_id", st.RealmID.String()),
				logger.ErrAttr(err),
			)
			failed++
			continue
		}
		logger.Info("mattermost auto sync finished",
			logger.StringAttr("realm_id", st.RealmID.String()),
			logger.Int32Attr("created", int32(result.Created)),
			logger.Int32Attr("linked", int32(result.Linked)),
			logger.Int32Attr("deactivated", int32(result.Deactivated)),
			logger.Int32Attr("reactivated", int32(result.Reactivated)),
		)
	}
	logger.Info("mattermost auto sync completed",
		logger.Int32Attr("realms", int32(len(settings))),
		logger.Int32Attr("failed", int32(failed)),
	)
	return nil
}

// syncRealmUsers — общее ядро синхронизации: сопоставляет пользователей
// Mattermost с системными (по mattermost_id, затем email, username и ФИО),
// создаёт недостающих и распространяет active/deactivated. actor идёт в аудит.
func (s *MattermostService) syncRealmUsers(ctx context.Context, realmID uuid.UUID, actor *models.Actor, teamNames []string) (*SyncUsersResult, error) {
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

	// GetAllUsers забирает из Mattermost только активных пользователей, поэтому
	// деактивированный в MM сотрудник оставался активным локально и продолжал
	// попадать в селекты заказчика/исполнителя. Отдельный запрос неактивных
	// нужен, чтобы синк снимал флаг у уже связанных учётных записей.
	// Ошибка не валит синк: импорт активных пользователей важнее.
	inactiveMM, err := s.most.Client.GetInactiveUsers(settings.BotToken)
	if err != nil {
		logger.Warn("failed to get inactive mattermost users", logger.ErrAttr(err))
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
		// Список по командам может включать деактивированных — для всех пользователей
		// их отсекает syncFilter в GetAllUsers, здесь проверка повторяется явно.
		if mmU.DeleteAt != 0 {
			continue
		}
		if _, ok := seen[mmU.Id]; ok {
			continue
		}
		seen[mmU.Id] = struct{}{}

		existing, existingErr := s.users.GetByMattermostID(ctx, mmU.Id)
		if existingErr == nil {
			// Возвращённого в MM сотрудника снова можно выбирать заказчиком:
			// локальный флаг должен подняться на той же синхронизации.
			if existing.Source.IsFromMattermost() && !existing.IsActive {
				if err := s.setLocalUserActive(ctx, existing, true, actor); err != nil {
					logger.Warn("failed to reactivate local mattermost user",
						logger.StringAttr("mm_user_id", mmU.Id),
						logger.StringAttr("user_id", existing.ID.String()),
						logger.ErrAttr(err),
					)
					result.Failed++
				} else {
					result.Reactivated++
				}
			}
			if err := s.ensureRealmMembership(ctx, &models.RealmMembershipDTO{
				RealmID:   realmID,
				UserID:    existing.ID,
				ActorID:   actor.ID,
				ActorName: actor.Name,
			}); err != nil {
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
				s.ensureLinkAndRealm(ctx, &models.RealmMembershipDTO{
					RealmID:   realmID,
					UserID:    sysU.ID,
					ActorID:   actor.ID,
					ActorName: actor.Name,
				}, mmU.Id, nil)
				result.Linked++
				matched = true
			}
		}

		if !matched && mmU.Username != "" {
			if sysU, ok := sysByUsername[strings.ToLower(mmU.Username)]; ok {
				s.ensureLinkAndRealm(ctx, &models.RealmMembershipDTO{
					RealmID:   realmID,
					UserID:    sysU.ID,
					ActorID:   actor.ID,
					ActorName: actor.Name,
				}, mmU.Id, nil)
				result.Linked++
				matched = true
			}
		}

		if !matched {
			if fio := buildFIO(mmU.FirstName, cleanMMLastName(mmU.LastName)); fio != "" {
				if sysU, ok := sysByFIO[fio]; ok {
					s.ensureLinkAndRealm(ctx, &models.RealmMembershipDTO{
						RealmID:   realmID,
						UserID:    sysU.ID,
						ActorID:   actor.ID,
						ActorName: actor.Name,
					}, mmU.Id, nil)
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
		if err := s.ensureRealmMembership(ctx, &models.RealmMembershipDTO{
			RealmID:   realmID,
			UserID:    newUserID,
			ActorID:   actor.ID,
			ActorName: actor.Name,
		}); err != nil {
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

	// Деактивированные в Mattermost: снимаем флаг только у локальных записей,
	// созданных из Mattermost. Пользователи Keycloak остаются под управлением
	// Keycloak — их активность перезаписывалась бы следующим keycloak-синком.
	seenInactive := make(map[string]struct{}, len(inactiveMM))
	for _, mmU := range inactiveMM {
		if mmU.Id == "" || mmU.DeleteAt == 0 {
			continue
		}
		if _, ok := seenInactive[mmU.Id]; ok {
			continue
		}
		seenInactive[mmU.Id] = struct{}{}

		existing, err := s.users.GetByMattermostID(ctx, mmU.Id)
		if err != nil {
			if !errors.Is(err, models.ErrNotFound) {
				logger.Warn("failed to find local user for inactive mattermost user",
					logger.StringAttr("mm_user_id", mmU.Id),
					logger.ErrAttr(err),
				)
				result.Failed++
			}
			continue
		}
		if !existing.Source.IsFromMattermost() || !existing.IsActive {
			continue
		}
		if err := s.setLocalUserActive(ctx, existing, false, actor); err != nil {
			logger.Warn("failed to deactivate local mattermost user",
				logger.StringAttr("mm_user_id", mmU.Id),
				logger.StringAttr("user_id", existing.ID.String()),
				logger.ErrAttr(err),
			)
			result.Failed++
			continue
		}
		result.Deactivated++
	}

	logger.Info("mattermost users synced",
		logger.StringAttr("realm_id", realmID.String()),
		logger.Int32Attr("created", int32(result.Created)),
		logger.Int32Attr("linked", int32(result.Linked)),
		logger.Int32Attr("failed", int32(result.Failed)),
		logger.Int32Attr("deactivated", int32(result.Deactivated)),
		logger.Int32Attr("reactivated", int32(result.Reactivated)),
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
