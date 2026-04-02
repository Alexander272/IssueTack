package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/enforcer"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

type PermissionService struct {
	repo     repository.Permissions
	enforcer *enforcer.EnforcerWrapper
	roles    Roles
}

func NewPermissionService(repo repository.Permissions, enforcer *enforcer.EnforcerWrapper, roles Roles) *PermissionService {
	return &PermissionService{
		repo:     repo,
		enforcer: enforcer,
		roles:    roles,
	}
}

type Permissions interface {
	GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error)
	Create(ctx context.Context, dto *models.PermissionDTO) error
	Delete(ctx context.Context, dto *models.DeletePermissionDTO) error
}

func (s *PermissionService) GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error) {
	data, err := s.repo.GetByRole(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions by role: %w", err)
	}
	return data, nil
}

func (s *PermissionService) Create(ctx context.Context, dto *models.PermissionDTO) error {
	err := s.repo.Create(ctx, dto)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}
	return nil
}

func (s *PermissionService) Delete(Ctx context.Context, dto *models.DeletePermissionDTO) error {
	err := s.repo.Delete(Ctx, dto)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}
	return nil
}

// // DTO для фронтенда
// type RolePermission struct {
// 	Object string `json:"object"` // например "task"
// 	Action string `json:"action"` // например "create"
// }

// type UserRoleAssignment struct {
// 	UserID   string `json:"user_id"`
// 	Role     string `json:"role"`
// 	Location string `json:"location"` // domain
// }

// func NewPermissionService(casbin *casbin.EnforcerWrapper, userRepo *repository.UserRepository) *PermissionService {
// 	return &PermissionService{casbin: casbin, userRepo: userRepo}
// }

// // AssignRoleToUser назначает роль пользователю на конкретной площадке
// func (s *PermissionService) AssignRoleToUser(ctx context.Context, assignment UserRoleAssignment) error {
// 	// 1. Проверка существования пользователя
// 	if exists, err := s.userRepo.Exists(ctx, assignment.UserID); err != nil || !exists {
// 		return errors.New("user not found")
// 	}

// 	// 2. Проверка существования локации (домена)
// 	// ... проверка через locationRepo ...

// 	// 3. Назначение в Casbin
// 	_, err := s.casbin.AddGroupingPolicy(assignment.UserID, assignment.Role, assignment.Location)
// 	return err
// }

// // CreateRolePermission создает новое право для роли
// func (s *PermissionService) CreateRolePermission(ctx context.Context, role, location string, perm RolePermission) error {
// 	// Валидация: объект и действие должны быть из белого списка,
// 	// чтобы админ не ввел случайную строку
// 	if !isValidObject(perm.Object) || !isValidAction(perm.Action) {
// 		return errors.New("invalid permission object or action")
// 	}

// 	_, err := s.casbin.AddPolicy(role, location, perm.Object, perm.Action)
// 	return err
// }

// // GetRoleMatrix возвращает данные для админки (Роли х Права)
// func (s *PermissionService) GetRoleMatrix(ctx context.Context, location string) (map[string][]RolePermission, error) {
// 	// Получаем все роли в этом домене (упрощенно)
// 	// В реальности нужно выгрузить все уникальные роли из политик
// 	roles := []string{"manager", "executor", "admin"}

// 	matrix := make(map[string][]RolePermission)

// 	for _, role := range roles {
// 		policies, err := s.casbin.GetPermissionsForRole(role, location)
// 		if err != nil {
// 			return nil, err
// 		}

// 		perms := make([]RolePermission, 0)
// 		for _, p := range policies {
// 			// p = [role, domain, object, action]
// 			if len(p) >= 4 {
// 				perms = append(perms, RolePermission{
// 					Object: p[2],
// 					Action: p[3],
// 				})
// 			}
// 		}
// 		matrix[role] = perms
// 	}
// 	return matrix, nil
// }

// func isValidObject(obj string) bool {
// 	allowed := map[string]bool{"task": true, "user": true, "location": true}
// 	return allowed[obj]
// }

// func isValidAction(act string) bool {
// 	allowed := map[string]bool{"create": true, "read": true, "update": true, "delete": true}
// 	return allowed[act]
// }

/*
// SyncRoleToCasbin — загружает все правила роли в Casbin
func (r *PermissionRepo) SyncRoleToCasbin(ctx context.Context, casbin *EnforcerWrapper, roleCode, domain string) error {
    // 1. Получаем все правила роли из БД
    perms, err := r.GetRolePermissionsByCode(ctx, roleCode, domain)
    if err != nil {
        return err
    }

    // 2. Для каждого правила добавляем в Casbin
    for _, perm := range perms {
        casbin.AddPolicy(roleCode, domain, perm.CasbinObject, perm.CasbinAction)
        // SavePolicy вызывается внутри AddPolicy вашей обёртки
    }

    return nil
}

// SyncAllRolesToCasbin — массовая синхронизация (при старте приложения)
func (r *PermissionRepo) SyncAllRolesToCasbin(ctx context.Context, casbin *EnforcerWrapper) error {
    // Загружаем все активные роли
    roles, _ := r.GetActiveRoles(ctx)

    for _, role := range roles {
        // Для root — даём полный доступ
        if role.Code == "root" {
            casbin.AddPolicy("root", "*", "*", "*")
            continue
        }

        // Для остальных — загружаем правила по доменам
        domains, _ := r.GetRoleDomains(ctx, role.ID)
        for _, domain := range domains {
            r.SyncRoleToCasbin(ctx, casbin, role.Code, domain)
        }
    }
    return nil
}
*/
