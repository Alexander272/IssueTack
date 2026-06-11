package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Nerzal/gocloak/v13"
	"github.com/google/uuid"
)

func (s *userService) Sync(ctx context.Context, actor *models.Actor) error {
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

	dbUsers, err := s.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch DB users: %w", err)
	}

	toCreate := make([]*models.UserDataDTO, 0)
	toUpdate := make([]*models.UserDataDTO, 0)
	toDelete := make([]uuid.UUID, 0)

	for _, dbU := range dbUsers {
		if kcData, exists := kcDataMap[dbU.ID]; exists {
			existUser := &models.UserDataDTO{
				ID:        dbU.ID,
				Username:  dbU.Username,
				FirstName: dbU.FirstName,
				LastName:  dbU.LastName,
				Email:     dbU.Email,
				IsActive:  dbU.IsActive,
			}
			if s.isChanged(existUser, kcData) {
				toUpdate = append(toUpdate, kcData)
			}
			delete(kcDataMap, dbU.ID)
		} else {
			toDelete = append(toDelete, dbU.ID)
		}
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
			"deleted", len(toDelete))
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

func (s *userService) collectSubGroupIDs(group *gocloak.Group) []string {
	if group == nil || group.ID == nil {
		return nil
	}
	ids := []string{*group.ID}
	s.collectNestedIDs(group.SubGroups, &ids)
	return ids
}

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

func (s *userService) mapToUserData(u *gocloak.User) *models.UserDataDTO {
	id, err := uuid.Parse(s.nonNil(u.ID))
	if err != nil {
		return nil
	}

	return &models.UserDataDTO{
		ID:        id,
		Username:  s.nonNil(u.Username),
		Email:     s.nonNil(u.Email),
		FirstName: s.nonNil(u.FirstName),
		LastName:  s.nonNil(u.LastName),
		IsActive:  u.Enabled != nil && *u.Enabled,
	}
}

func (s *userService) nonNil(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *userService) isChanged(old, new *models.UserDataDTO) bool {
	return old.Username != new.Username ||
		old.Email != new.Email ||
		old.FirstName != new.FirstName ||
		old.LastName != new.LastName ||
		old.IsActive != new.IsActive
}
