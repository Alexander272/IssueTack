package services

import (
	"context"
	"fmt"
	"log"
	"slices"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/casbin/casbin/v3"
)

type accessPolicesService struct {
	enforcer casbin.IEnforcer
	adapter  Adapter
	eventBus *events.PolicyEventManager
	quit     chan struct{}
}

type PoliciesDeps struct {
	Conf     config.CasbinConfig
	Adapter  Adapter
	EventBus *events.PolicyEventManager
	Cache    SessionCacher
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
		eventBus: deps.EventBus,
		quit:     make(chan struct{}),
	}

	updateChan := deps.EventBus.Subscribe()
	go func() {
		defer deps.EventBus.Unsubscribe(updateChan)
		for {
			select {
			case <-s.quit:
				return
			case _, ok := <-updateChan:
				if !ok {
					return
				}
				logger.Info("Received policy update event, reloading...")
				deps.Cache.Flush(context.Background())
				if err := s.enforcer.LoadPolicy(); err != nil {
					logger.Warn("failed to reload policy after event", "error", err.Error())
				}
			}
		}
	}()

	return s
}

type AccessPolices interface {
	Enforce(sub, dom, obj, act string) (bool, error)
	Reload() error
	GetPolicies(user, domain string) (*models.Access, error)
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

func (s *accessPolicesService) Close() {
	close(s.quit)
}

func (s *accessPolicesService) GetPolicies(user, domain string) (*models.Access, error) {
	allPermissions, err := s.enforcer.GetImplicitPermissionsForUser(user, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get implicit permissions for user: %w", err)
	}

	permsMap := make(map[string]bool)
	var role string

	for _, p := range allPermissions {
		// Сохраняем первую роль (или последнюю - зависит от логики Casbin)
		if role == "" && len(p) > 0 && p[0] != "" {
			role = p[0]
		}

		// Пропускаем некорректные правила
		if len(p) < 4 {
			continue
		}

		resource, action := p[2], p[3]

		// Формируем правило в нужном формате
		var rule string
		if resource == "*" && action == "*" {
			rule = "*:*"
		} else if action == "*" {
			rule = fmt.Sprintf("%s:*", resource)
		} else if resource == "*" {
			rule = fmt.Sprintf("*:%s", action)
		} else {
			rule = fmt.Sprintf("%s:%s", resource, action)
		}

		permsMap[rule] = true
	}

	// Конвертируем map в slice
	perms := make([]string, 0, len(permsMap))
	for rule := range permsMap {
		perms = append(perms, rule)
	}

	slices.Sort(perms)

	return &models.Access{
		Role:   role,
		Domain: domain,
		Perms:  perms,
	}, nil
}
