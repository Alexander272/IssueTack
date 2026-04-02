package services

import "github.com/Alexander272/IssueTrack/backend/internal/enforcer"

type AccessPolicesService struct {
	enforcer      *enforcer.EnforcerWrapper
	roles         Roles
	roleHierarchy RoleHierarchy
	permissions   Permissions
}

type PoliciesDeps struct {
	Enforcer      *enforcer.EnforcerWrapper
	Roles         Roles
	RoleHierarchy RoleHierarchy
	Permissions   Permissions
}

func NewAccessPoliciesService(deps *PoliciesDeps) *AccessPolicesService {
	return &AccessPolicesService{
		enforcer:      deps.Enforcer,
		roles:         deps.Roles,
		roleHierarchy: deps.RoleHierarchy,
		permissions:   deps.Permissions,
	}
}

// init_permissions.go
// func SyncAll(pool *pgxpool.Pool, casbin *EnforcerWrapper) error {
//     ctx := context.Background()

//     // 1. Очищаем старые g-политики, связанные с ролями (но не user→role!)
//     // Опционально: если хотите полный ресинк
//     // casbin.ClearGPolicy() // осторожно: удалит и user→role связи

//     // 2. Загружаем все активные роли
//     roles, err := GetActiveRoles(ctx, pool)
//     if err != nil {
//         return err
//     }

//     for _, role := range roles {
//         // === Обработка root ===
//         if role.Code == "root" {
//             casbin.AddPolicy("root", "*", "*", "*")
//             continue
//         }

//         // === Прямые права роли (из role_permissions) ===
//         perms, err := GetRolePermissions(ctx, pool, role.ID)
//         if err != nil {
//             continue
//         }
//         for _, perm := range perms {
//             casbin.AddPolicy(role.Code, perm.Domain, perm.CasbinObject, perm.CasbinAction)
//         }

//         // === Наследование (роль → родительские роли) ===
//         inheritance, err := GetRoleInheritance(ctx, pool, role.ID)
//         if err != nil {
//             continue
//         }
//         for _, inh := range inheritance {
//             // g(дочерняя, родительская, домен)
//             casbin.AddGroupingPolicy(role.Code, inh.ParentRoleCode, inh.Domain)
//         }
//     }

//     return nil
// }
