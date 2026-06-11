package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
)

type adapterService struct {
	ctx           context.Context
	perms         Permissions
	roleHierarchy RoleHierarchy
	users         Users
}

type AdapterDeps struct {
	Permissions   Permissions
	RoleHierarchy RoleHierarchy
	Users         Users
	Ctx           context.Context
}

func NewAdapter(deps *AdapterDeps) *adapterService {
	return &adapterService{
		ctx:           deps.Ctx,
		perms:         deps.Permissions,
		roleHierarchy: deps.RoleHierarchy,
		users:         deps.Users,
	}
}

type Adapter interface {
	LoadPolicy(model model.Model) error

	SavePolicy(model model.Model) error
	AddPolicy(sec string, ptype string, rule []string) error
	RemovePolicy(sec string, ptype string, rule []string) error
	RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error
}

func (s *adapterService) LoadPolicy(model model.Model) error {
	logger.Info("load policy")

	rootPolicy := "p, root, *, *, *"
	if err := persist.LoadPolicyLine(rootPolicy, model); err != nil {
		return fmt.Errorf("failed to load root policy: %w", err)
	}

	permissions, err := s.perms.LoadPolicy(s.ctx)
	if err != nil {
		return err
	}
	for _, p := range permissions {
		line := fmt.Sprintf("p, %s, %s, %s, %s", p.Role, p.Realm, p.Object, p.Action)
		logger.Debug("permissions", logger.StringAttr("item", line))
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("failed to load permissions policy. error: %w", err)
		}
	}

	roles, err := s.roleHierarchy.LoadPolicy(s.ctx)
	if err != nil {
		return err
	}
	for _, r := range roles {
		line := fmt.Sprintf("g, %s, %s, %s", r.ParentRole, r.Role, r.Realm)
		logger.Debug("permissions", logger.StringAttr("group", line))
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("failed to load group policy. error: %w", err)
		}
	}

	users, err := s.users.LoadPolicy(s.ctx)
	if err != nil {
		return err
	}
	for _, u := range users {
		line := fmt.Sprintf("g, %s, %s, %s", u.UserID, u.RoleName, u.Realm)
		logger.Debug("permissions", logger.StringAttr("group", line))
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("failed to load group policy. error: %w", err)
		}
	}

	return nil
}

func (s *adapterService) SavePolicy(model model.Model) error {
	return nil
}

func (s *adapterService) AddPolicy(sec string, ptype string, rule []string) error {
	return nil
}

func (s *adapterService) RemovePolicy(sec string, ptype string, rule []string) error {
	return nil
}

func (s *adapterService) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return nil
}
