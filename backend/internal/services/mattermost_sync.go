package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

// resolveOrCreateUser находит или создаёт системного пользователя по userID
// из Mattermost, возвращая его UUID и имя. Сначала ищет уже сохранённую связку;
// если её нет — пытается сопоставить с существующим пользователем (по email,
// username или ФИО) и при неудаче создаёт нового. Реализовано так, чтобы
// внешний Mattermost-пользователь всегда мог создавать заявки без ручной
// регистрации в системе.
func (s *MattermostService) resolveOrCreateUser(ctx context.Context, realmID uuid.UUID, mmUserID string, siteID *uuid.UUID) (uuid.UUID, string, error) {
	existing, err := s.users.GetByMattermostID(ctx, mmUserID)
	if err == nil {
		s.ensureLinkAndRealm(ctx, realmID, existing.ID, mmUserID, siteID, existing.ID, existing.Username)
		return existing.ID, existing.Username, nil
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to get realm settings: %w", err)
	}

	mmUser, err := s.most.Client.GetUser(settings.BotToken, mmUserID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to get mattermost user: %w", err)
	}

	sysUsers, err := s.users.GetAll(ctx, nil)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to get system users: %w", err)
	}

	if matched, userID, username := matchByEmail(sysUsers, mmUser.Email); matched {
		s.ensureLinkAndRealm(ctx, realmID, userID, mmUser.Id, siteID, userID, username)
		return userID, username, nil
	}

	if matched, userID, username := matchByUsername(sysUsers, mmUser.Username); matched {
		s.ensureLinkAndRealm(ctx, realmID, userID, mmUser.Id, siteID, userID, username)
		return userID, username, nil
	}

	mmFio := buildFIO(mmUser.FirstName, mmUser.LastName)
	if mmFio != "" {
		for _, sysU := range sysUsers {
			if buildFIO(sysU.FirstName, sysU.LastName) == mmFio {
				s.ensureLinkAndRealm(ctx, realmID, sysU.ID, mmUser.Id, siteID, sysU.ID, sysU.Username)
				return sysU.ID, sysU.Username, nil
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
		LastName:     mmUser.LastName,
		Email:        mmUser.Email,
		IsActive:     true,
	}
	if err := s.users.CreateSeveral(ctx, nil, []*models.UserDataDTO{userDTO}); err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to create user: %w", err)
	}
	s.ensureRealmMembership(ctx, newUserID, realmID, newUserID, mmUser.Username)

	logger.Info("created user from mattermost",
		logger.StringAttr("mm_user_id", mmUserID),
		logger.StringAttr("user_id", newUserID.String()),
	)
	return newUserID, mmUser.Username, nil
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

// handleSync — обработчик команды «синхронизировать [команды]»: сопоставляет
// пользователей Mattermost с системными (по email/username/ФИО) и создаёт
// недостающих локальных пользователей в указанном realm. Доступна только
// администратору realm; необязательными аргументами можно ограничить синк
// конкретными командами Mattermost.
func (s *MattermostService) handleSync(ctx context.Context, settings *models.RealmMattermost, senderMmID string, message string) error {
	senderID, senderName, err := s.resolveOrCreateUser(ctx, settings.RealmID, senderMmID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve sender: %w", err)
	}

	senderRealm, err := s.userRealms.GetByUserAndRealm(ctx, senderID, settings.RealmID)
	if err != nil || senderRealm.Role == nil || senderRealm.Role.Slug != "admin" {
		if err := s.most.DM.Send(settings.BotToken, settings.BotUserID, senderMmID,
			"Только администраторы могут синхронизировать пользователей"); err != nil {
			return fmt.Errorf("failed to send no-permission message: %w", err)
		}
		return nil
	}

	parts := syncCommands.FindStringSubmatch(message)
	teamNames := []string{}
	if len(parts) > 1 && parts[1] != "" {
		for _, t := range strings.Split(parts[1], ",") {
			if t = strings.TrimSpace(t); t != "" {
				teamNames = append(teamNames, t)
			}
		}
	}

	mmUsers, err := s.fetchMMUsers(settings.BotToken, teamNames)
	if err != nil {
		return err
	}

	sysUsers, err := s.users.GetAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get system users: %w", err)
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

	seen := make(map[string]struct{}, len(mmUsers))
	created := 0
	linked := 0

	for _, mmU := range mmUsers {
		if _, ok := seen[mmU.Id]; ok {
			continue
		}
		seen[mmU.Id] = struct{}{}

		existing, existingErr := s.users.GetByMattermostID(ctx, mmU.Id)
		if existingErr == nil {
			if err := s.ensureRealmMembership(ctx, existing.ID, settings.RealmID, senderID, senderName); err != nil {
				logger.Warn("failed to add user to realm",
					logger.StringAttr("mm_user_id", mmU.Id),
					logger.ErrAttr(err),
				)
			} else {
				linked++
			}
			continue
		}

		matched := false

		if mmU.Email != "" {
			if sysU, ok := sysByEmail[strings.ToLower(mmU.Email)]; ok {
				s.ensureLinkAndRealm(ctx, settings.RealmID, sysU.ID, mmU.Id, nil, senderID, senderName)
				linked++
				matched = true
			}
		}

		if !matched && mmU.Username != "" {
			if sysU, ok := sysByUsername[strings.ToLower(mmU.Username)]; ok {
				s.ensureLinkAndRealm(ctx, settings.RealmID, sysU.ID, mmU.Id, nil, senderID, senderName)
				linked++
				matched = true
			}
		}

		if !matched {
			if fio := buildFIO(mmU.FirstName, mmU.LastName); fio != "" {
				if sysU, ok := sysByFIO[fio]; ok {
					s.ensureLinkAndRealm(ctx, settings.RealmID, sysU.ID, mmU.Id, nil, senderID, senderName)
					linked++
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
			LastName:     mmU.LastName,
			Email:        mmU.Email,
			IsActive:     true,
		}
		if err := s.users.CreateSeveral(ctx, nil, []*models.UserDataDTO{userDTO}); err != nil {
			logger.Warn("failed to create user from mattermost",
				logger.StringAttr("mm_user_id", mmU.Id),
				logger.ErrAttr(err),
			)
			continue
		}
		if err := s.ensureRealmMembership(ctx, newUserID, settings.RealmID, senderID, senderName); err != nil {
			logger.Warn("failed to add user to realm",
				logger.StringAttr("mm_user_id", mmU.Id),
				logger.ErrAttr(err),
			)
		}
		created++
	}

	msg := fmt.Sprintf("Синхронизация завершена. Создано: %d, привязано: %d", created, linked)
	if err := s.most.DM.Send(settings.BotToken, settings.BotUserID, senderMmID, msg); err != nil {
		bestEffortError("failed to send sync result", err, map[string]string{"mm_user_id": senderMmID})
	}
	return nil
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
