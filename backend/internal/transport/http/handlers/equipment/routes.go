package equipment

import (
	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
)

// Deps — зависимости хендлеров оборудования. Собираются в вызывающем коде
// именованными полями: позиционный список из семи аргументов легко перепутать,
// а два последних сервиса ещё и необязательны (могут быть nil).
type Deps struct {
	Service        services.Equipment
	Importer       EquipmentImporter
	MaxSize        int64
	Middleware     *middleware.Middleware
	PeriodicTasks  *services.EquipmentPeriodicTaskService
	OperationTasks *services.EquipmentOperationTaskService
}

// Register собирает дерево маршрутов оборудования. Регистрация разнесена по
// тематическим файлам, но уровни доступа задаются здесь же, тремя явными
// группами: раньше они нанизывались через equipment.Use(...) и порядок вызовов
// register* молча определял, под какими правами окажется маршрут.
func Register(api *gin.RouterGroup, deps Deps) {
	h := NewHandler(deps)

	read := api.Group("/equipment", deps.Middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Read()))
	write := read.Group("", deps.Middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Write()))
	del := write.Group("", deps.Middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Delete()))

	registerUnit(read, write, del, h)
	registerComposition(read, write, h)
	registerTypes(read, write, del, h)
	registerTemplates(read, write, del, h)
	registerActs(read, write, h)
	registerAssignment(read, write, h)
	registerImport(read, write, h)
	registerPeriodicTasks(read, write, deps)
	registerOperationTasks(read, write, deps)
}
