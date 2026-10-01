package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Nerzal/gocloak/v13"
	"github.com/google/uuid"
)

var internalNumberRe = regexp.MustCompile(`^(.*?)\s*\((\d*)\)$`)

// Sync синхронизирует пользователей из Keycloak с локальной БД: создаёт новых, обновляет изменённых и удаляет отсутствующих.
//
// force снимает защиту от удаления пользователей, у которых есть заявки (см. usersWithTickets).
// Без него синхронизация останавливается с SY001: таких пользователей нет в группе
// Keycloak, но tickets.creator_id/owner_id объявлены как ON DELETE CASCADE, поэтому
// синк удалил бы их вместе с заявками. Проверка по факту наличия заявок, а не по
// source/mattermost_id, потому что после бэкфилла связанный с Mattermost сотрудник
// остаётся source=keycloak и в Keycloak не состоит — привязка к Mattermost не
// защищает от удаления, а вот заявки защищают.
func (s *userService) Sync(ctx context.Context, actor *models.Actor, force bool) error {
	logger.Info("Sync users started")

	token, err := s.keycloak.Login(ctx)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	group, err := s.keycloak.Client.GetGroupByPath(ctx, token.AccessToken, s.keycloak.Realm, "/"+s.keycloak.GroupName)
	if err != nil {
		return fmt.Errorf("failed to get group by path: %w", err)
	}
	if group.ID == nil {
		return fmt.Errorf("group ID is nil for group '%s'", s.keycloak.GroupName)
	}

	allGroupIDs := s.collectSubGroupIDs(group)
	if allGroupIDs == nil {
		logger.Info(fmt.Sprintf("group '%s' and sub-groups are empty", s.keycloak.GroupName))
		return nil
	}

	userMap := make(map[string]*gocloak.User)
	for _, gid := range allGroupIDs {
		members, err := s.getAllGroupMembers(ctx, token.AccessToken, gid)
		if err != nil {
			return fmt.Errorf("failed to get group members for group %s: %w", gid, err)
		}
		for _, m := range members {
			if m.ID != nil {
				userMap[*m.ID] = m
			}
		}
	}

	if len(userMap) == 0 {
		logger.Info(fmt.Sprintf("group '%s' and sub-groups are empty", s.keycloak.GroupName))
		return nil
	}

	kcDataMap := make(map[uuid.UUID]*models.UserDataDTO, len(userMap))
	for _, u := range userMap {
		if u.Enabled != nil && !*u.Enabled {
			continue
		}

		userData := s.mapToUserData(u)
		if userData == nil {
			continue
		}
		kcDataMap[userData.ID] = userData
	}

	dbUsers, err := s.GetAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch DB users: %w", err)
	}

	toCreate := make([]*models.UserDataDTO, 0)
	toUpdate := make([]*models.UserDataDTO, 0)
	toDelete := make([]uuid.UUID, 0)
	deleteUsernames := make(map[uuid.UUID]string, 0)

	for _, dbU := range dbUsers {
		if protectedFromKeycloakSync(dbU) {
			continue
		}
		if kcData, exists := kcDataMap[dbU.ID]; exists {
			existUser := &models.UserDataDTO{
				ID:             dbU.ID,
				Username:       dbU.Username,
				FirstName:      dbU.FirstName,
				LastName:       dbU.LastName,
				Email:          dbU.Email,
				IsActive:       dbU.IsActive,
				InternalNumber: dbU.InternalNumber,
			}
			if s.isChanged(existUser, kcData) {
				toUpdate = append(toUpdate, kcData)
			}
			delete(kcDataMap, dbU.ID)
		} else {
			logger.Debug("user not found in Keycloak, will be deleted",
				"user_id", dbU.ID,
				"username", dbU.Username,
				"email", dbU.Email,
			)
			toDelete = append(toDelete, dbU.ID)
			deleteUsernames[dbU.ID] = dbU.Username
		}
	}

	usersWithTickets := make([]string, 0)
	if len(toDelete) > 0 {
		ids, err := s.repo.GetIDsWithTickets(ctx, toDelete)
		if err != nil {
			return fmt.Errorf("failed to check users with tickets: %w", err)
		}
		for _, id := range ids {
			if username, ok := deleteUsernames[id]; ok {
				usersWithTickets = append(usersWithTickets, username)
			}
		}
	}

	if len(usersWithTickets) > 0 && !force {
		logger.Warn("keycloak sync stopped: users with tickets would be deleted",
			"count", len(usersWithTickets),
			"usernames", strings.Join(usersWithTickets, ", "))
		return models.ErrKeycloakSyncLinkedUsers(usersWithTickets, len(usersWithTickets))
	}

	for _, newU := range kcDataMap {
		toCreate = append(toCreate, newU)
	}

	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if len(toCreate) > 0 {
			if err := s.CreateSeveral(ctx, tx, toCreate); err != nil {
				return err
			}
		}
		if len(toUpdate) > 0 {
			if err := s.UpdateSeveral(ctx, tx, toUpdate); err != nil {
				return err
			}
		}
		if len(toDelete) > 0 {
			if err := s.DeleteSeveral(ctx, tx, toDelete); err != nil {
				return err
			}
		}

		logger.Info("Sync finished",
			"created", len(toCreate),
			"updated", len(toUpdate),
			"deleted", len(toDelete),
			"deleted_with_tickets", len(usersWithTickets))
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to execute batch: %w", err)
	}

	event := events.PolicyEvent{
		ChangedBy:     actor.ID,
		ChangedByName: actor.Name,
		Action:        "sync_users",
		EntityType:    "users",
	}
	s.eventBus.Notify(event)
	return nil
}

// getAllGroupMembers постранично (по 1000 записей) выгружает всех участников группы из Keycloak,
// пока ответ не станет короче размера страницы. Постраничный цикл нужен, чтобы не терять участников
// в больших группах, превышающих лимит одного ответа Keycloak.
func (s *userService) getAllGroupMembers(ctx context.Context, token, groupID string) ([]*gocloak.User, error) {
	var all []*gocloak.User
	first := 0
	max := 1000

	for {
		params := gocloak.GetGroupsParams{
			First: &first,
			Max:   &max,
		}
		members, err := s.keycloak.Client.GetGroupMembers(ctx, token, s.keycloak.Realm, groupID, params)
		if err != nil {
			return nil, err
		}
		all = append(all, members...)
		if len(members) < max {
			break
		}
		first += max
	}
	return all, nil
}

// collectSubGroupIDs возвращает плоский список ID группы и всех её вложенных подгрупп.
// Синхронизация идёт по всей ветке, а не только по корневой группе, поэтому подгруппы разворачиваются
// рекурсивно до любого уровня вложенности.
func (s *userService) collectSubGroupIDs(group *gocloak.Group) []string {
	if group == nil || group.ID == nil {
		return nil
	}
	ids := []string{*group.ID}
	s.collectNestedIDs(group.SubGroups, &ids)
	return ids
}

// collectNestedIDs рекурсивно обходит произвольную вложенность подгрупп и накапливает их ID в общий список,
// пропуская подгруппы без идентификатора.
func (s *userService) collectNestedIDs(subGroups *[]gocloak.Group, ids *[]string) {
	if subGroups == nil {
		return
	}
	for _, sg := range *subGroups {
		if sg.ID == nil {
			continue
		}
		*ids = append(*ids, *sg.ID)
		s.collectNestedIDs(sg.SubGroups, ids)
	}
}

// extractInternalNumber извлекает внутренний номер, привязанный к фамилии пользователя в формате
// "Фамилия (12345)". Если формат не совпадает, возвращает фамилию как есть и пустой номер,
// чтобы не терять данные при ручном вводе без номера.
func extractInternalNumber(lastName string) (string, string) {
	m := internalNumberRe.FindStringSubmatch(lastName)
	if m == nil {
		return lastName, ""
	}
	return m[1], m[2]
}

// mapToUserData преобразует пользователя Keycloak в DTO для локальной БД, отбрасывая пользователей
// с некорректным UUID (возвращает nil), чтобы они не сломали последующую пакетную вставку.
// Поля Keycloak разворачиваются через nonNil, т.к. API отдаёт указатели.
func (s *userService) mapToUserData(u *gocloak.User) *models.UserDataDTO {
	id, err := uuid.Parse(s.nonNil(u.ID))
	if err != nil {
		return nil
	}

	lastName, internalNumber := extractInternalNumber(s.nonNil(u.LastName))

	return &models.UserDataDTO{
		ID:             id,
		Username:       s.nonNil(u.Username),
		Email:          s.nonNil(u.Email),
		FirstName:      s.nonNil(u.FirstName),
		LastName:       lastName,
		IsActive:       u.Enabled != nil && *u.Enabled,
		InternalNumber: internalNumber,
	}
}

// nonNil разворачивает строковый указатель в значение; Keycloak возвращает пустые поля как nil,
// а для записи в БД удобнее всегда иметь обычную строку.
func (s *userService) nonNil(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// protectedFromKeycloakSync сообщает, что пользователя синхронизация с Keycloak
// не трогает: он остаётся в базе, даже если в Keycloak его нет.
//
//   - Системные пользователи исключены по прямому условию (историческое поведение).
//   - Пользователи из Mattermost в Keycloak не состоят: у них нет ни учётной записи,
//     ни id = Keycloak sub, поэтому в каталоге они не найдутся никогда. Удалять их
//     нельзя — tickets.creator_id объявлен как ON DELETE CASCADE, и синхронизация
//     унесла бы заявки заявителей вместе с комментариями и вложениями.
//
// Этот.source — первая линия защиты, но она держится только на бэкфилле:
// после того как SyncRealmUsers свяжет сотрудника с Mattermost, его source навсегда
// остаётся keycloak, а mattermost_id заполняется. Поэтому уход такого сотрудника из
// Keycloak защита выше не видит — за это отвечает отдельная проверка по заявкам
// (GetIDsWithTickets, см. Sync и ErrKeycloakSyncLinkedUsers).
func protectedFromKeycloakSync(u *models.UserData) bool {
	return u.IsSystem || u.Source.IsFromMattermost()
}

// isChanged сравнивает пользователя из БД с данными из Keycloak. Синхронизация идемпотентна:
// обновляются только те записи, где хотя бы одно поле реально отличается, чтобы не перезаписывать
// записи впустую на каждом прогоне.
func (s *userService) isChanged(old, new *models.UserDataDTO) bool {
	return old.Username != new.Username ||
		old.Email != new.Email ||
		old.FirstName != new.FirstName ||
		old.LastName != new.LastName ||
		old.IsActive != new.IsActive ||
		old.InternalNumber != new.InternalNumber
}
