package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
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
}

func (s *adapterService) LoadPolicy(model model.Model) error {
	return s.loadPolicy(model, nil)
}

func (s *adapterService) LoadFilteredPolicy(model model.Model, req *models.GetPoliciesDTO) error {
	return s.loadPolicy(model, req)
}

func (s *adapterService) loadPolicy(model model.Model, req *models.GetPoliciesDTO) error {
	rootPolicy := "p, root, *, *, *"
	if err := persist.LoadPolicyLine(rootPolicy, model); err != nil {
		return fmt.Errorf("failed to load root policy: %w", err)
	}

	// load permissions
	permissions, err := s.perms.LoadPolicy(context.Background(), req)
	if err != nil {
		return err
	}
	for _, p := range permissions {
		line := fmt.Sprintf("p, %s, %s, %s, %s", p.Role, p.Realm, p.Object, p.Action)
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
		line := fmt.Sprintf("g, %s, %s, %s", r.Role, r.ParentRole, r.Realm)
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
		line := fmt.Sprintf("g, %s, %s, %s", u.UserID, u.RoleName, u.Realm)
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("failed to load group policy. error: %w", err)
		}
	}

	return nil
}
