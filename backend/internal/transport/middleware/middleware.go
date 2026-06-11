package middleware

import (
	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
)

type Middleware struct {
	keycloak *auth.KeycloakClient
	auth     *config.AuthConfig
	services *services.Services
}

func NewMiddleware(services *services.Services, auth *config.AuthConfig, keycloak *auth.KeycloakClient) *Middleware {
	return &Middleware{
		keycloak: keycloak,
		auth:     auth,
		services: services,
	}
}
