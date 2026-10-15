package equipment

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// actDocxMimeType — тип docx по RFC 6838: тот же, что у файлов Word.
const actDocxMimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

type Handler struct {
	service services.Equipment
	// maxSize — предел размера подписанного акта: multipart отбрасывается еще на
	// парсинге тела, иначе большой файл успел бы лечь в память/на диск.
	maxSize int64
}

func NewHandler(service services.Equipment, maxSize int64) *Handler {
	return &Handler{service: service, maxSize: maxSize}
}

func Register(api *gin.RouterGroup, service services.Equipment, maxSize int64, middleware *middleware.Middleware) {
	handlers := NewHandler(service, maxSize)

	equipment := api.Group("/equipment", middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Read()))
	{
		equipment.GET("", handlers.getAll)
		equipment.GET("/:id", handlers.getByID)
		equipment.GET("/:id/children", handlers.getChildren)
		equipment.GET("/:id/assignments", handlers.getAssignmentHistory)
		equipment.GET("/:id/events", handlers.getEvents)
		equipment.GET("/:id/composition", handlers.getComposition)
		equipment.GET("/:id/composition/history", handlers.getCompositionHistory)
		equipment.GET("/tree", handlers.getTree)
		equipment.GET("/stats", handlers.getStats)
		equipment.GET("/types", handlers.getTypes)
		equipment.GET("/types/:id", handlers.getTypeByID)
		equipment.GET("/acts", handlers.getActs)
		equipment.GET("/acts/:id", handlers.getActByID)
		equipment.GET("/acts/:id/document", handlers.getActDocument)
		equipment.GET("/acts/:id/signed", handlers.getActSignedFile)

		equipment.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Write()))
		equipment.POST("", handlers.create)
		equipment.PUT("/:id", handlers.update)
		equipment.POST("/:id/assign", handlers.assign)
		equipment.POST("/:id/unassign", handlers.unassign)
		equipment.POST("/:id/status", handlers.changeStatus)
		equipment.POST("/:id/composition", handlers.addComponents)
		equipment.DELETE("/:id/composition", handlers.removeComponents)
		equipment.POST("/types", handlers.createType)
		equipment.PUT("/types/:id", handlers.updateType)
		equipment.POST("/acts", handlers.createAct)
		equipment.POST("/acts/:id/signed", handlers.uploadActSigned)

		equipment.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Delete()))
		equipment.DELETE("/:id", handlers.delete)
		equipment.DELETE("/types/:id", handlers.deleteType)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	filter := &models.EquipmentFilter{}
	if err := c.ShouldBindQuery(filter); err != nil {
		response.SendError(c, err)
		return
	}
	filter.RealmID = realmID

	data, total, err := h.service.Get(c, filter)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, total)
}

func (h *Handler) getByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetByID(c, &models.GetEquipmentByIdDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) getChildren(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetChildren(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getTree(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.GetEquipmentTreeDTO{}
	if err := c.ShouldBindQuery(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	data, total, err := h.service.GetTree(c, dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, total)
}

func (h *Handler) getStats(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetStats(c, realmID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) create(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.RealmID = realmID
	dto.CreatedBy = &actor.ID
	dto.Actor = actor

	data, err := h.service.Create(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: data.ID, Message: "Оборудование создано"})
}

func (h *Handler) update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if dto.ID == nil || id != *dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = &id
	dto.RealmID = realmID
	dto.Actor = actor

	if _, err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Оборудование обновлено"})
}

func (h *Handler) delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c, &models.DelEquipmentDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) getTypes(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetTypes(c, &models.GetEquipmentTypesDTO{RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getTypeByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetTypeByID(c, &models.GetEquipmentTypeByIdDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) createType(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentTypeDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	if err := h.service.CreateType(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Тип оборудования создан"})
}

func (h *Handler) updateType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentTypeDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if dto.ID == nil || id != *dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	dto.ID = &id
	dto.RealmID = realmID

	if err := h.service.UpdateType(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Тип оборудования обновлён"})
}

func (h *Handler) deleteType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteType(c, &models.DelEquipmentTypeDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) assign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.AssignEquipmentDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = id
	dto.RealmID = realmID
	dto.Actor = actor

	if err := h.service.Assign(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Оборудование выдано"})
}

func (h *Handler) unassign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.UnassignEquipmentDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = id
	dto.RealmID = realmID
	dto.Actor = actor

	if err := h.service.Unassign(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Оборудование возвращено"})
}

func (h *Handler) changeStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.ChangeStatusDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = id
	dto.RealmID = realmID
	dto.Actor = actor

	data, err := h.service.ChangeStatus(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) getAssignmentHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetAssignmentHistory(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getEvents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	filter := &models.EquipmentEventsFilter{}
	if err := c.ShouldBindQuery(filter); err != nil {
		response.SendError(c, err)
		return
	}

	data, err := h.service.GetEvents(c, realmID, id, filter.Limit, filter.Offset)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getComposition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetComposition(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getCompositionHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetCompositionHistory(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) addComponents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.KitCompositionOpDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = id
	dto.RealmID = realmID
	dto.Actor = actor

	if err := h.service.AddComponents(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Состав комплекта обновлён"})
}

func (h *Handler) removeComponents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.KitCompositionOpDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = id
	dto.RealmID = realmID
	dto.Actor = actor

	if err := h.service.RemoveComponents(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Состав комплекта обновлён"})
}

func (h *Handler) getActs(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	filter := &models.EquipmentActsFilter{}
	if err := c.ShouldBindQuery(filter); err != nil {
		response.SendError(c, err)
		return
	}
	filter.RealmID = realmID

	data, total, err := h.service.GetActs(c, filter)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, total)
}

func (h *Handler) getActByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetActByID(c, &models.GetEquipmentActByIDDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) createAct(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentActDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.RealmID = realmID
	dto.Actor = actor

	data, err := h.service.CreateAct(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: data.ID, Message: "Акт сформирован"})
}

// getActDocument отдаёт docx акта, собранный по снимку состава. Заголовок
// выгрузки — attachment: документ выгружают, чтобы распечатать и подписать.
func (h *Handler) getActDocument(c *gin.Context) {
	id, realmID, ok := actRequestParams(c)
	if !ok {
		return
	}

	doc, err := h.service.GetActDocument(c, &models.GetEquipmentActByIDDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}

	c.Header("Content-Disposition", actDisposition("attachment", doc.FileName))
	c.Data(http.StatusOK, actDocxMimeType, doc.Content)
}

// getActSignedFile отдаёт загруженный подписанный акт. Заголовок — inline:
// файл открывают в браузере (просмотр), а скачивание инициирует клиент.
func (h *Handler) getActSignedFile(c *gin.Context) {
	id, realmID, ok := actRequestParams(c)
	if !ok {
		return
	}

	act, file, err := h.service.GetActSignedFile(c, &models.GetEquipmentActByIDDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	defer file.Close()

	mimeType := "application/octet-stream"
	if act.SignedMimeType != nil && *act.SignedMimeType != "" {
		mimeType = *act.SignedMimeType
	}
	var size int64 = -1
	if act.SignedFileSize != nil {
		size = *act.SignedFileSize
	}
	name := act.Number
	if act.SignedFileName != nil && *act.SignedFileName != "" {
		name = *act.SignedFileName
	}

	c.Header("Content-Disposition", actDisposition("inline", name))
	c.DataFromReader(http.StatusOK, size, mimeType, file, nil)
}

// uploadActSigned принимает скан или фото подписанного акта.
func (h *Handler) uploadActSigned(c *gin.Context) {
	id, realmID, ok := actRequestParams(c)
	if !ok {
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	// К пределу сервиса добавляем запас на multipart-служебные поля (~64 КБ),
	// чтобы тело с файлом допустимого размера не отбрасывалось на парсинге.
	if h.maxSize > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxSize+(64<<10))
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	defer file.Close()

	dto := &models.SignEquipmentActDTO{
		ID:       id,
		RealmID:  realmID,
		FileName: header.Filename,
		FileSize: header.Size,
		MimeType: header.Header.Get("Content-Type"),
		File:     file,
		Actor:    actor,
	}
	if err := h.service.SignAct(c, dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Акт подписан"})
}

// actRequestParams разбирает id акта из пути и реалм из заголовка. Акты ищутся
// парой (id, realm): без домена акт другого реалма открывался бы по прямому id.
func actRequestParams(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return uuid.Nil, uuid.Nil, false
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return id, realmID, true
}

// actDisposition собирает Content-Disposition для файла акта. Номер акта
// кириллический, поэтому рядом с ASCII-фолбэком идёт filename* по RFC 5987:
// без него браузер сохранит файл под именем из URL.
func actDisposition(kind, fileName string) string {
	fallback := strings.Map(func(r rune) rune {
		if r < 128 && r != '"' && r != '\\' {
			return r
		}
		return '_'
	}, fileName)

	return fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`, kind, fallback, url.PathEscape(fileName))
}
