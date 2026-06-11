package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
)

type adapterService struct {
	perms         Permissions
	roleHierarchy RoleHierarchy
	users         Users
}

type AdapterDeps struct {
	Permissions   Permissions
	RoleHierarchy RoleHierarchy
	Users         Users
}

func NewAdapter(deps *AdapterDeps) *adapterService {
	return &adapterService{
		perms:         deps.Permissions,
		roleHierarchy: deps.RoleHierarchy,
		users:         deps.Users,
	}
}

type Adapter interface {
	LoadPolicy(model model.Model) error
	LoadFilteredPolicy(model model.Model, req *models.GetPoliciesDTO) error

	SavePolicy(model model.Model) error
	AddPolicy(sec string, ptype string, rule []string) error
	RemovePolicy(sec string, ptype string, rule []string) error
	RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error
}

func (s *adapterService) LoadPolicy(model model.Model) error {
	return s.loadPolicy(model, nil)
}

func (s *adapterService) LoadFilteredPolicy(model model.Model, req *models.GetPoliciesDTO) error {
	return s.loadPolicy(model, req)
}

func (s *adapterService) loadPolicy(model model.Model, req *models.GetPoliciesDTO) error {
	rootPolicy := "p, root, *, *"
	if err := persist.LoadPolicyLine(rootPolicy, model); err != nil {
		return fmt.Errorf("failed to load root policy: %w", err)
	}

	// load permissions
	permissions, err := s.perms.LoadPolicy(context.Background(), req)
	if err != nil {
		return err
	}
	for _, p := range permissions {
		line := fmt.Sprintf("p, %s, %s, %s", p.Role, p.Object, p.Action)
		logger.Debug("permissions", logger.StringAttr("item", line))
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("failed to load policy. error: %w", err)
		}
	}

	// load role hierarchy
	roles, err := s.roleHierarchy.LoadPolicy(context.Background(), req)
	if err != nil {
		return err
	}
	for _, r := range roles {
		line := fmt.Sprintf("g, %s, %s", r.ParentRole, r.Role)
		logger.Debug("permissions", logger.StringAttr("group", line))
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("failed to load group policy. error: %w", err)
		}
	}

	//load user roles
	users, err := s.users.LoadPolicy(context.Background(), req)
	if err != nil {
		return err
	}
	for _, u := range users {
		line := fmt.Sprintf("g, %s, %s", u.UserID, u.RoleName)
		logger.Debug("permissions", logger.StringAttr("group", line))
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("failed to load group policy. error: %w", err)
		}
	}

	return nil
}

// SavePolicy saves all policy rules to the storage.
func (s *adapterService) SavePolicy(model model.Model) error {
	return nil
}

// AddPolicy adds a policy rule to the storage.
// This is part of the Auto-Save feature.
func (s *adapterService) AddPolicy(sec string, ptype string, rule []string) error {
	return nil
}

// RemovePolicy removes a policy rule from the storage.
// This is part of the Auto-Save feature.
func (s *adapterService) RemovePolicy(sec string, ptype string, rule []string) error {
	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
// This is part of the Auto-Save feature.
func (a *adapterService) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return nil
}
