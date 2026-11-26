package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLocationDeleteRepo — заглушка справочника расположений для проверки
// запрета удаления используемой записи. Остальные методы приходят через
// внедрённый интерфейс и в этом тесте не вызываются.
type fakeLocationDeleteRepo struct {
	repository.SiteLocations
	used    bool
	deleted bool
}

func (f *fakeLocationDeleteRepo) HasReferences(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.used, nil
}

func (f *fakeLocationDeleteRepo) Delete(_ context.Context, _ *models.DelSiteLocationDTO) error {
	f.deleted = true
	return nil
}

func TestSiteService_DeleteLocation(t *testing.T) {
	dto := &models.DelSiteLocationDTO{ID: uuid.New(), SiteID: uuid.New()}

	t.Run("used location is rejected", func(t *testing.T) {
		locations := &fakeLocationDeleteRepo{used: true}
		svc := NewSiteService(nil, locations)

		err := svc.DeleteLocation(context.Background(), dto)

		require.ErrorIs(t, err, models.ErrSiteLocationInUse)
		assert.False(t, locations.deleted)
	})

	t.Run("unused location is deleted", func(t *testing.T) {
		locations := &fakeLocationDeleteRepo{}
		svc := NewSiteService(nil, locations)

		err := svc.DeleteLocation(context.Background(), dto)

		require.NoError(t, err)
		assert.True(t, locations.deleted)
	})
}
