package services

import (
	"context"
	"fmt"
	"log"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/google/uuid"
)

// SessionService управляет сессиями пользователей: вход, выход, обновление токенов и получение прав.
type SessionService struct {
	keycloak  *auth.KeycloakClient
	userRealm UserRealms
	user      Users
	groups    Groups
	policies  AccessPolicies
	cache     SessionCacher
}

// NewSessionService создаёт сервис сессий с заданными зависимостями.
func NewSessionService(keycloak *auth.KeycloakClient, policies AccessPolicies, userRealm UserRealms, user Users, groups Groups, cache SessionCacher) *SessionService {
	return &SessionService{
		keycloak:  keycloak,
		policies:  policies,
		userRealm: userRealm,
		user:      user,
		groups:    groups,
		cache:     cache,
	}
}

// Session описывает сервис управления сессиями пользователей.
type Session interface {
	SignIn(ctx context.Context, u models.SignIn) (*models.User, error)
	SignOut(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (*models.User, error)
	DecodeAccessToken(ctx context.Context, token string) (*models.User, error)
	GetAllCapabilities(ctx context.Context, userID uuid.UUID) (map[string]*models.UserCapabilities, error)
}

// SignIn выполняет вход пользователя через Keycloak и возвращает данные пользователя с токенами.
func (s *SessionService) SignIn(ctx context.Context, u models.SignIn) (*models.User, error) {
	res, err := s.keycloak.Client.Login(ctx, s.keycloak.ClientId, s.keycloak.ClientSecret, s.keycloak.Realm, u.Username, u.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to login to keycloak. error: %w", err)
	}

	user, err := s.DecodeAccessToken(ctx, res.AccessToken)
	if err != nil {
		return nil, err
	}

	if err := s.loadUserRealms(ctx, user); err != nil {
		return nil, err
	}

	user.AccessToken = res.AccessToken
	user.RefreshToken = res.RefreshToken

	return user, nil
}

// SignOut завершает сессию пользователя в Keycloak по refresh-токену.
func (s *SessionService) SignOut(ctx context.Context, refreshToken string) error {
	err := s.keycloak.Client.Logout(ctx, s.keycloak.ClientId, s.keycloak.ClientSecret, s.keycloak.Realm, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to logout to keycloak. error: %w", err)
	}
	return nil
}

// Refresh обновляет токены сессии по refresh-токену и возвращает актуальные данные пользователя.
func (s *SessionService) Refresh(ctx context.Context, refreshToken string) (*models.User, error) {
	res, err := s.keycloak.Client.RefreshToken(ctx, refreshToken, s.keycloak.ClientId, s.keycloak.ClientSecret, s.keycloak.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token in keycloak. error: %w", err)
	}

	user, err := s.DecodeAccessToken(ctx, res.AccessToken)
	if err != nil {
		return nil, err
	}

	if err := s.loadUserRealms(ctx, user); err != nil {
		return nil, err
	}

	user.AccessToken = res.AccessToken
	user.RefreshToken = res.RefreshToken

	return user, nil
}

// loadUserRealms подгружает в пользователя список его realm'ов и тут же наполняет capabilities.
// Вызывается после входа/обновления токена, чтобы в вернувшемся пользователе всегда был полный контекст
// (realm'ы + права/группы) без дополнительного запроса со стороны клиента.
func (s *SessionService) loadUserRealms(ctx context.Context, user *models.User) error {
	userRealms, err := s.userRealm.GetByUserID(ctx, user.ID)
	if err != nil {
		return err
	}
	user.Realms = userRealms

	s.loadUserCapabilities(ctx, user)

	return nil
}

// loadUserCapabilities собирает для каждого realm'а пользователя список управляемых и членских групп,
// а также признак того, что пользователь является администратором realm (realm-wide пермишены области).
// Ошибки получения групп здесь не фатальны: при сбое группа считается пустой, чтобы вход пользователя
// не падал из-за второстепенных данных.
func (s *SessionService) loadUserCapabilities(ctx context.Context, user *models.User) {
	caps := make(map[string]*models.UserCapabilities)
	for _, r := range user.Realms {
		realmID := r.RealmID
		realmIDStr := realmID.String()

		managedGroups, err := s.groups.GetManagedGroups(ctx, user.ID, &realmID)
		if err != nil {
			log.Printf("WARN: failed to get managed groups for user %s realm %s: %v", user.ID, realmIDStr, err)
			managedGroups = nil
		}

		memberGroups, err := s.groups.GetMemberGroups(ctx, user.ID, &realmID)
		if err != nil {
			log.Printf("WARN: failed to get member groups for user %s realm %s: %v", user.ID, realmIDStr, err)
			memberGroups = nil
		}

		isRealmAdmin, err := isRealmSupervisor(s.policies, user.ID, realmIDStr)
		if err != nil {
			log.Printf("WARN: failed to check realm supervisor for user %s realm %s: %v", user.ID, realmIDStr, err)
			isRealmAdmin = false
		}

		caps[realmIDStr] = &models.UserCapabilities{
			ManagedGroupIDs: managedGroups,
			MemberGroupIDs:  memberGroups,
			IsRealmAdmin:    isRealmAdmin,
		}
	}
	user.Capabilities = caps
}

// GetAllCapabilities возвращает возможности пользователя по каждому realm'у: управляемые и членские группы, признак администратора realm.
func (s *SessionService) GetAllCapabilities(ctx context.Context, userID uuid.UUID) (map[string]*models.UserCapabilities, error) {
	userRealms, err := s.userRealm.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user realms: %w", err)
	}

	caps := make(map[string]*models.UserCapabilities)
	for _, r := range userRealms {
		realmID := r.RealmID
		realmIDStr := realmID.String()

		managedGroups, err := s.groups.GetManagedGroups(ctx, userID, &realmID)
		if err != nil {
			return nil, fmt.Errorf("failed to get managed groups for realm %s: %w", realmIDStr, err)
		}

		memberGroups, err := s.groups.GetMemberGroups(ctx, userID, &realmID)
		if err != nil {
			return nil, fmt.Errorf("failed to get member groups for realm %s: %w", realmIDStr, err)
		}

		isRealmAdmin, err := isRealmSupervisor(s.policies, userID, realmIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to check realm supervisor for realm %s: %w", realmIDStr, err)
		}

		caps[realmIDStr] = &models.UserCapabilities{
			ManagedGroupIDs: managedGroups,
			MemberGroupIDs:  memberGroups,
			IsRealmAdmin:    isRealmAdmin,
		}
	}
	return caps, nil
}

// DecodeAccessToken декодирует access-токен, собирает данные пользователя и его права доступа по realm'ам.
func (s *SessionService) DecodeAccessToken(ctx context.Context, token string) (*models.User, error) {
	_, claims, err := s.keycloak.Client.DecodeAccessToken(ctx, token, s.keycloak.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to decode access token. error: %w", err)
	}

	c := *claims

	username, ok := c["preferred_username"].(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("missing or invalid preferred_username in token")
	}
	userIDStr, ok := c["sub"].(string)
	if !ok || userIDStr == "" {
		return nil, fmt.Errorf("missing or invalid sub in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user id. error: %w", err)
	}

	user := &models.User{
		ID:   userID,
		Name: username,
	}

	if perms := s.cache.Get(ctx, userIDStr); perms != nil {
		user.Permissions = perms
		return user, nil
	}

	userRealms, err := s.userRealm.GetByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user realms: %w", err)
	}
	user.Permissions = map[string][]string{}
	for _, r := range userRealms {
		access, err := s.policies.GetPolicies(userIDStr, r.RealmID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get policies: %w", err)
		}
		user.Permissions[r.RealmID.String()] = access.Perms
	}
	s.cache.Set(ctx, userIDStr, user.Permissions)

	return user, nil
}
