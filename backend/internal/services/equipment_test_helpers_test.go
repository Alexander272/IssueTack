package services

import (
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
)

// testEquipmentConf — конфиг файлового хранилища для тестов сервиса: каталог
// загрузок во временной директории, чтобы подписанные акты не падали в рабочий
// uploads. Отдельный хелпер нужен потому, что сервис оборудования держит conf
// обязательным: без него падают не только тесты подписания, но и конструктор.
func testEquipmentConf(t *testing.T) *config.FileServerConfig {
	t.Helper()
	return &config.FileServerConfig{UploadDir: t.TempDir(), MaxSize: 10 << 20}
}
