package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/google/uuid"
)

type SessionService struct {
	keycloak  *auth.KeycloakClient
	userRealm UserRealms
	user      Users
	policies  AccessPolices
	cache     SessionCacher
}

func NewSessionService(keycloak *auth.KeycloakClient, policies AccessPolices, userRealm UserRealms, user Users, cache SessionCacher) *SessionService {
	return &SessionService{
		keycloak:  keycloak,
		policies:  policies,
		userRealm: userRealm,
		user:      user,
		cache:     cache,
	}
}

type Session interface {
	SignIn(ctx context.Context, u models.SignIn) (*models.User, error)
	SignOut(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (*models.User, error)
	DecodeAccessToken(ctx context.Context, token string) (*models.User, error)
}

func (s *SessionService) SignIn(ctx context.Context, u models.SignIn) (*models.User, error) {
	res, err := s.keycloak.Client.Login(ctx, s.keycloak.ClientId, s.keycloak.ClientSecret, s.keycloak.Realm, u.Username, u.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to login to keycloak. error: %w", err)
	}

	user, err := s.DecodeAccessToken(ctx, res.AccessToken)
	if err != nil {
		return nil, err
	}

	// if err := s.loadUserPermissions(ctx, user); err != nil {
	// 	return nil, err
	// }

	user.AccessToken = res.AccessToken
	user.RefreshToken = res.RefreshToken

	return user, nil
}

func (s *SessionService) SignOut(ctx context.Context, refreshToken string) error {
	err := s.keycloak.Client.Logout(ctx, s.keycloak.ClientId, s.keycloak.ClientSecret, s.keycloak.Realm, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to logout to keycloak. error: %w", err)
	}
	return nil
}

func (s *SessionService) Refresh(ctx context.Context, refreshToken string) (*models.User, error) {
	res, err := s.keycloak.Client.RefreshToken(ctx, refreshToken, s.keycloak.ClientId, s.keycloak.ClientSecret, s.keycloak.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token in keycloak. error: %w", err)
	}

	user, err := s.DecodeAccessToken(ctx, res.AccessToken)
	if err != nil {
		return nil, err
	}

	// if err := s.loadUserPermissions(ctx, user); err != nil {
	// 	return nil, err
	// }

	user.AccessToken = res.AccessToken
	user.RefreshToken = res.RefreshToken

	return user, nil
}

// func (s *SessionService) loadUserPermissions(ctx context.Context, user *models.User) error {
// 	userRealms, err := s.userRealm.GetByUserID(ctx, user.ID)
// 	if err != nil {
// 		return err
// 	}
// 	user.Realms = userRealms

// 	user.Permissions = map[string][]string{}
// 	for _, r := range userRealms {
// 		access, err := s.policies.GetPolicies(user.ID.String(), r.RealmID.String())
// 		if err != nil {
// 			return err
// 		}
// 		user.Permissions[r.RealmID.String()] = access.Perms
// 	}

// 	s.cache.Set(ctx, user.ID.String(), user.Permissions)
// 	return nil
// }

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
