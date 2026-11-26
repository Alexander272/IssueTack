package services

import (
	"context"
	"sort"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// testEquipmentConf — конфиг файлового хранилища для тестов сервиса: каталог
// загрузок во временной директории, чтобы подписанные акты не падали в рабочий
// uploads. Отдельный хелпер нужен потому, что сервис оборудования держит conf
// обязательным: без него падают не только тесты подписания, но и конструктор.
func testEquipmentConf(t *testing.T) *config.FileServerConfig {
	t.Helper()
	return &config.FileServerConfig{UploadDir: t.TempDir(), MaxSize: 10 << 20}
}

// fakeSiteLocationsRepo — минимальный справочник расположений для тестов
// оборудования. Остальные методы интерфейса не нужны, поэтому они приходят
// через внедрённый nil-интерфейс и при вызове сразу покажут ошибку теста.
type fakeSiteLocationsRepo struct {
	repository.SiteLocations
	names map[uuid.UUID]string
	sites map[uuid.UUID]uuid.UUID
}

func newFakeSiteLocations() *fakeSiteLocationsRepo {
	return &fakeSiteLocationsRepo{
		names: map[uuid.UUID]string{},
		sites: map[uuid.UUID]uuid.UUID{},
	}
}

func (f *fakeSiteLocationsRepo) add(siteID, locationID uuid.UUID, name string) {
	f.names[locationID] = name
	f.sites[locationID] = siteID
}

func (f *fakeSiteLocationsRepo) GetName(_ context.Context, siteID, locationID uuid.UUID) (string, error) {
	if f.sites[locationID] != siteID {
		return "", models.ErrNotFound
	}
	return f.names[locationID], nil
}

func (f *fakeSiteLocationsRepo) GetBySite(_ context.Context, siteID uuid.UUID) ([]*models.SiteLocation, error) {
	data := []*models.SiteLocation{}
	for id, site := range f.sites {
		if site == siteID {
			data = append(data, &models.SiteLocation{ID: id, SiteID: siteID, Name: f.names[id]})
		}
	}
	sort.Slice(data, func(i, j int) bool { return data[i].Name < data[j].Name })
	return data, nil
}
