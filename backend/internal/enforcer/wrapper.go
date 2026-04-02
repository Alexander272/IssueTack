package enforcer

import (
	"sync"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/casbin/casbin/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EnforcerWrapper struct {
	mu       sync.RWMutex
	enforcer *casbin.Enforcer
}

// NewEnforcerWrapper создает обертку
func NewEnforcerWrapper(conf config.CasbinConfig, pool *pgxpool.Pool) (*EnforcerWrapper, error) {
	e, err := NewEnforcer(conf, pool)
	if err != nil {
		return nil, err
	}

	return &EnforcerWrapper{enforcer: e}, nil
}

// Enforce проверяет права (читаемая операция)
func (w *EnforcerWrapper) Enforce(sub, dom, obj, act string) (bool, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.enforcer.Enforce(sub, dom, obj, act)
}

// AddGroupingPolicy назначает роль пользователю
func (w *EnforcerWrapper) AddGroupingPolicy(user, role, domain string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ok, err := w.enforcer.AddGroupingPolicy(user, role, domain)
	if err != nil {
		return false, err
	}

	// При включенном AutoSave это сохранится в БД автоматически
	// Но мы явно вызываем SavePolicy для надежности
	if err := w.enforcer.SavePolicy(); err != nil {
		return false, err
	}

	return ok, nil
}

// RemoveGroupingPolicy удаляет роль у пользователя
func (w *EnforcerWrapper) RemoveGroupingPolicy(user, role, domain string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ok, err := w.enforcer.RemoveGroupingPolicy(user, role, domain)
	if err != nil {
		return false, err
	}

	if err := w.enforcer.SavePolicy(); err != nil {
		return false, err
	}

	return ok, nil
}

// AddPolicy добавляет разрешение для роли
func (w *EnforcerWrapper) AddPolicy(role, domain, obj, act string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ok, err := w.enforcer.AddPolicy(role, domain, obj, act)
	if err != nil {
		return false, err
	}

	if err := w.enforcer.SavePolicy(); err != nil {
		return false, err
	}

	return ok, nil
}

// RemovePolicy удаляет разрешение для роли
func (w *EnforcerWrapper) RemovePolicy(role, domain, obj, act string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ok, err := w.enforcer.RemovePolicy(role, domain, obj, act)
	if err != nil {
		return false, err
	}

	if err := w.enforcer.SavePolicy(); err != nil {
		return false, err
	}

	return ok, nil
}

// GetRolesForUserInDomain получает роли пользователя в домене
func (w *EnforcerWrapper) GetRolesForUserInDomain(user, domain string) []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.enforcer.GetRolesForUserInDomain(user, domain)
}

// GetUsersForRoleInDomain получает пользователей роли в домене
func (w *EnforcerWrapper) GetUsersForRoleInDomain(role, domain string) []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.enforcer.GetUsersForRoleInDomain(role, domain)
}

// GetPermissionsForRole получает все права роли в домене
func (w *EnforcerWrapper) GetPermissionsForRole(role, domain string) ([][]string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.enforcer.GetFilteredPolicy(0, role, domain)
}

// GetAllRoles получает все уникальные роли
func (w *EnforcerWrapper) GetAllRoles() ([]string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.enforcer.GetAllRoles()
}

// ReloadPolicy принудительно перезагружает политики из БД
// Вызывается автоматически через Watcher при изменении в других инстансах
func (w *EnforcerWrapper) ReloadPolicy() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enforcer.LoadPolicy()
}
