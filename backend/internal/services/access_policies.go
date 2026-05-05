package services

import (
	"fmt"
	"log"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/casbin/casbin/v3"
)

type accessPolicesService struct {
	enforcer casbin.IEnforcer
	adapter  Adapter
	eventBus *events.PolicyEventManager
}

type PoliciesDeps struct {
	Conf     config.CasbinConfig
	Adapter  Adapter
	EventBus *events.PolicyEventManager
}

func NewAccessPoliciesService(deps *PoliciesDeps) *accessPolicesService {
	enforcer, err := casbin.NewEnforcer(deps.Conf.ModelPath, deps.Adapter)
	if err != nil {
		log.Fatalf("failed to initialize permission service. error: %s", err.Error())
	}

	if err = enforcer.LoadPolicy(); err != nil {
		log.Fatalf("failed to load policy from DB: %s", err.Error())
	}

	s := &accessPolicesService{
		enforcer: enforcer,
		adapter:  deps.Adapter,
	}

	go func() {
		updateChan := deps.EventBus.Subscribe()
		for range updateChan {
			log.Println("Received policy update event, reloading...")
			s.enforcer.LoadPolicy()
		}
	}()

	return s
}

type AccessPolices interface {
	Enforce(sub, dom, obj, act string) (bool, error)
	Reload() error
}

func (s *accessPolicesService) Enforce(sub, dom, obj, act string) (bool, error) {
	return s.enforcer.Enforce(sub, dom, obj, act)
}

func (s *accessPolicesService) Reload() error {
	err := s.enforcer.LoadPolicy()
	if err != nil {
		return fmt.Errorf("failed to reload policies: %w", err)
	}
	return nil
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
