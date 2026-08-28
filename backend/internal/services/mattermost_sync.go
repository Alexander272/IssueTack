package services

import (
	"context"
	"fmt"
	"strings"

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
		ur, urErr := s.userRealms.GetByUserAndRealm(ctx, existing.ID, realmID)
		if urErr != nil || ur == nil {
			roleID, roleErr := s.roles.GetIDBySlug(ctx, realmID, "user")
			if roleErr == nil {
				_ = s.userRealms.CreateSeveral(ctx, nil, []*models.UserRealmDTO{
					{UserID: existing.ID, RealmID: realmID, RoleID: &roleID, IsActive: true},
				})
			}
		}
		s.ensureLinkAndRealm(ctx, realmID, existing.ID, mmUserID, siteID)
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
		s.ensureLinkAndRealm(ctx, realmID, userID, mmUser.Id, siteID)
		return userID, username, nil
	}

	if matched, userID, username := matchByUsername(sysUsers, mmUser.Username); matched {
		s.ensureLinkAndRealm(ctx, realmID, userID, mmUser.Id, siteID)
		return userID, username, nil
	}

	mmFio := buildFIO(mmUser.FirstName, mmUser.LastName)
	if mmFio != "" {
		for _, sysU := range sysUsers {
			if buildFIO(sysU.FirstName, sysU.LastName) == mmFio {
				s.ensureLinkAndRealm(ctx, realmID, sysU.ID, mmUser.Id, siteID)
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
	roleID, roleErr := s.roles.GetIDBySlug(ctx, realmID, "user")
	if roleErr == nil {
		_ = s.userRealms.CreateSeveral(ctx, nil, []*models.UserRealmDTO{
			{UserID: newUserID, RealmID: realmID, RoleID: &roleID, IsActive: true},
		})
	}

	logger.Info("created user from mattermost",
		logger.StringAttr("mm_user_id", mmUserID),
		logger.StringAttr("user_id", newUserID.String()),
	)
	return newUserID, mmUser.Username, nil
}

// ensureLinkAndRealm привязывает системного пользователя к его Mattermost
// userID (через users.mattermost_id) и добавляет его в realm с ролью «user»,
// если он там ещё не состоит. Также обновляет site_id, если он передан.
// Ошибки некритичны (только логируются), чтобы не прерывать основной поток
// сопоставления пользователя.
func (s *MattermostService) ensureLinkAndRealm(ctx context.Context, realmID uuid.UUID, userID uuid.UUID, mmUserID string, siteID *uuid.UUID) {
	mmCopy := mmUserID
	if err := s.users.UpdateMMAndSite(ctx, nil, &models.UserDataDTO{
		ID:           userID,
		MattermostID: &mmCopy,
		SiteID:       siteID,
	}); err != nil {
		logger.Warn("failed to update mattermost id/site", logger.ErrAttr(err))
	}
	ur, urErr := s.userRealms.GetByUserAndRealm(ctx, userID, realmID)
	if urErr != nil || ur == nil {
		roleID, roleErr := s.roles.GetIDBySlug(ctx, realmID, "user")
		if roleErr == nil {
			_ = s.userRealms.CreateSeveral(ctx, nil, []*models.UserRealmDTO{
				{UserID: userID, RealmID: realmID, RoleID: &roleID, IsActive: true},
			})
		}
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
	senderID, _, err := s.resolveOrCreateUser(ctx, settings.RealmID, senderMmID, nil)
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

	roleID, err := s.roles.GetIDBySlug(ctx, settings.RealmID, "user")
	if err != nil {
		return fmt.Errorf("failed to get default role: %w", err)
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
			ur, urErr := s.userRealms.GetByUserAndRealm(ctx, existing.ID, settings.RealmID)
			if urErr != nil || ur == nil {
				roleIDCopy := roleID
				if err := s.userRealms.CreateSeveral(ctx, nil, []*models.UserRealmDTO{
					{UserID: existing.ID, RealmID: settings.RealmID, RoleID: &roleIDCopy, IsActive: true},
				}); err != nil {
					logger.Warn("failed to add user to realm",
						logger.StringAttr("mm_user_id", mmU.Id),
						logger.ErrAttr(err),
					)
				} else {
					linked++
				}
			}
			continue
		}

		matched := false

		if mmU.Email != "" {
			if sysU, ok := sysByEmail[strings.ToLower(mmU.Email)]; ok {
				s.ensureLinkAndRealm(ctx, settings.RealmID, sysU.ID, mmU.Id, nil)
				linked++
				matched = true
			}
		}

		if !matched && mmU.Username != "" {
			if sysU, ok := sysByUsername[strings.ToLower(mmU.Username)]; ok {
				s.ensureLinkAndRealm(ctx, settings.RealmID, sysU.ID, mmU.Id, nil)
				linked++
				matched = true
			}
		}

		if !matched {
			if fio := buildFIO(mmU.FirstName, mmU.LastName); fio != "" {
				if sysU, ok := sysByFIO[fio]; ok {
					s.ensureLinkAndRealm(ctx, settings.RealmID, sysU.ID, mmU.Id, nil)
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
		roleIDCopy := roleID
		_ = s.userRealms.CreateSeveral(ctx, nil, []*models.UserRealmDTO{
			{UserID: newUserID, RealmID: settings.RealmID, RoleID: &roleIDCopy, IsActive: true},
		})
		created++
	}

	msg := fmt.Sprintf("Синхронизация завершена. Создано: %d, привязано: %d", created, linked)
	if err := s.most.DM.Send(settings.BotToken, settings.BotUserID, senderMmID, msg); err != nil {
		return fmt.Errorf("failed to send sync result: %w", err)
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
