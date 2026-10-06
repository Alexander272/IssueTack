package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_EnsureRealmMembership(t *testing.T) {
	mockUserRealm := new(MockUserRealmsService)
	svc := &userService{userRealm: mockUserRealm}

	userID := uuid.New()
	realmID := uuid.New()

	mockUserRealm.On("GetByUserAndRealm", mock.Anything, userID, realmID).
		Return(&models.UserRealm{UserID: userID, RealmID: realmID}, nil)

	assert.NoError(t, svc.ensureRealmMembership(context.Background(), userID, realmID))
}

// Сотрудник, не состоящий в авторизованном реалме, недоступен для правки учётки:
// иначе users:write в своём realm менял бы ФИО/email чужого пользователя.
func TestUserService_EnsureRealmMembership_ForeignUser(t *testing.T) {
	mockUserRealm := new(MockUserRealmsService)
	svc := &userService{userRealm: mockUserRealm}

	userID := uuid.New()
	realmID := uuid.New()

	mockUserRealm.On("GetByUserAndRealm", mock.Anything, userID, realmID).
		Return(nil, models.ErrNotFound)

	err := svc.ensureRealmMembership(context.Background(), userID, realmID)
	assert.ErrorIs(t, err, models.ErrNotFound)
}

// Без realm запроса проверка не выполняется: это внутренние вызовы (синхронизация
// Keycloak, создание пользователя), где домен ещё не задан.
func TestUserService_EnsureRealmMembership_NoRealm(t *testing.T) {
	mockUserRealm := new(MockUserRealmsService)
	svc := &userService{userRealm: mockUserRealm}

	assert.NoError(t, svc.ensureRealmMembership(context.Background(), uuid.New(), uuid.Nil))
	mockUserRealm.AssertNotCalled(t, "GetByUserAndRealm", mock.Anything, mock.Anything, mock.Anything)
}
