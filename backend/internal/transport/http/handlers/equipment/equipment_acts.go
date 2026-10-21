package equipment

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// actDocxMimeType — тип docx по RFC 6838: тот же, что у файлов Word.
const actDocxMimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

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
