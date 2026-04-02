package services

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"your-project/internal/authz"
// 	"your-project/internal/models"
// 	"your-project/internal/repository"
// )

// type RBACService struct {
// 	authz          *authz.EnforcerWrapper
// 	roleRepo       *repository.RoleRepository
// 	locationRepo   *repository.LocationRepository
// 	permissionRepo *repository.PermissionRepository
// }

// func NewRBACService(
// 	authz *authz.EnforcerWrapper,
// 	roleRepo *repository.RoleRepository,
// 	locationRepo *repository.LocationRepository,
// 	permissionRepo *repository.PermissionRepository,
// ) *RBACService {
// 	return &RBACService{
// 		authz:          authz,
// 		roleRepo:       roleRepo,
// 		locationRepo:   locationRepo,
// 		permissionRepo: permissionRepo,
// 	}
// }

// // === Управление ролями (бизнес-логика) ===

// func (s *RBACService) CreateRole(ctx context.Context, role *models.Role) error {
// 	// 1. Валидация: имя роли уникально
// 	if exists, err := s.roleRepo.RoleExists(ctx, role.Name); err != nil || exists {
// 		return errors.New("role already exists")
// 	}

// 	// 2. Создаем роль в бизнес-таблице
// 	if err := s.roleRepo.Create(ctx, role); err != nil {
// 		return fmt.Errorf("failed to create role in DB: %w", err)
// 	}

// 	// 3. (Опционально) Создаем базовые права для роли в Casbin
// 	// Например, новая роль по умолчанию не имеет прав — это безопасно
// 	return nil
// }

// // === Назначение прав (Casbin + валидация) ===

// func (s *RBACService) AddPermissionToRole(ctx context.Context, roleName, locationCode, object, action string) error {
// 	// 1. Валидация: роль существует в бизнес-таблице
// 	if exists, err := s.roleRepo.RoleExists(ctx, roleName); err != nil || !exists {
// 		return errors.New("role not found")
// 	}

// 	// 2. Валидация: локация существует
// 	if exists, err := s.locationRepo.ExistsByCode(ctx, locationCode); err != nil || !exists {
// 		return errors.New("location not found")
// 	}

// 	// 3. Валидация: разрешение есть в каталоге (защита от опечаток)
// 	if valid, err := s.permissionRepo.IsValid(ctx, object, action); err != nil || !valid {
// 		return fmt.Errorf("invalid permission: %s/%s", object, action)
// 	}

// 	// 4. Добавляем политику в Casbin (через обертку)
// 	_, err := s.authz.AddPolicy(roleName, locationCode, object, action)
// 	return err
// }

// // === Назначение роли пользователю ===

// func (s *RBACService) AssignRoleToUser(ctx context.Context, userID, roleName, locationCode string) error {
// 	// 1. Валидация всех сущностей
// 	if exists, err := s.roleRepo.RoleExists(ctx, roleName); err != nil || !exists {
// 		return errors.New("role not found")
// 	}
// 	if exists, err := s.locationRepo.ExistsByCode(ctx, locationCode); err != nil || !exists {
// 		return errors.New("location not found")
// 	}
// 	// ... проверка пользователя ...

// 	// 2. Назначаем роль в Casbin
// 	_, err := s.authz.AddGroupingPolicy(userID, roleName, locationCode)
// 	return err
// }

// // === Чтение для админки ===

// type RolePermissionsView struct {
// 	RoleName    string              `json:"role_name"`
// 	Location    string              `json:"location"`
// 	Permissions []models.Permission `json:"permissions"`
// }

// func (s *RBACService) GetRolePermissionsView(ctx context.Context, roleName, locationCode string) (*RolePermissionsView, error) {
// 	// Читаем права напрямую из Casbin (быстро, т.к. в памяти)
// 	policies, err := s.authz.GetPermissionsForRole(roleName, locationCode)
// 	if err != nil {
// 		return nil, err
// 	}

// 	permissions := make([]models.Permission, 0, len(policies))
// 	for _, p := range policies {
// 		if len(p) >= 4 {
// 			permissions = append(permissions, models.Permission{
// 				Object: p[2],
// 				Action: p[3],
// 			})
// 		}
// 	}

// 	return &RolePermissionsView{
// 		RoleName:    roleName,
// 		Location:    locationCode,
// 		Permissions: permissions,
// 	}, nil
// }

// // GetAllRolesWithLocations возвращает список ролей с информацией,
// // на каких площадках они назначены пользователям
// func (s *RBACService) GetAllRolesWithLocations(ctx context.Context) ([]*models.RoleInfo, error) {
// 	// 1. Получаем роли из бизнес-таблицы (с описанием, метаданными)
// 	roles, err := s.roleRepo.GetAll(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 2. Для каждой роли получаем информацию из Casbin
// 	result := make([]*models.RoleInfo, 0, len(roles))
// 	for _, role := range roles {
// 		info := &models.RoleInfo{
// 			ID:          role.ID,
// 			Name:        role.Name,
// 			Description: role.Description,
// 			Locations:   make([]string, 0),
// 		}

// 		// Получаем всех пользователей с этой ролью на всех локациях
// 		// (упрощенно: берем уникальные домены из политик)
// 		allPolicies, _ := s.authz.GetFilteredPolicy(0, role.Name)          // p-type политики
// 		allGroupings, _ := s.authz.GetFilteredGroupingPolicy(0, role.Name) // g-type назначения

// 		locationSet := make(map[string]bool)
// 		for _, p := range allPolicies {
// 			if len(p) >= 2 {
// 				locationSet[p[1]] = true
// 			}
// 		}
// 		for _, g := range allGroupings {
// 			if len(g) >= 3 {
// 				locationSet[g[2]] = true
// 			}
// 		}

// 		for loc := range locationSet {
// 			info.Locations = append(info.Locations, loc)
// 		}

// 		result = append(result, info)
// 	}

// 	return result, nil
// }
