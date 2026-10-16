package mattermost

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestHandleWSEvent_IgnoresSystemPost закрепляет регресс «бот написал тому, кто
// ничего не писал»: системные сообщения Mattermost («X добавил бота в канал»)
// приходят от имени пользователя и не пусты, поэтому до фильтра они доходили до
// HandleDM и получали справку. service здесь намеренно nil — если фильтр не
// сработает, обработчик дойдёт до сервиса и упадёт, а тест это покажет.
func TestHandleWSEvent_IgnoresSystemPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{service: nil}
	engine := gin.New()
	engine.POST("/event", h.handleWSEvent)

	body := `{"event":"posted","data":{"post":"{\"id\":\"p1\",\"user_id\":\"mm1\",\"channel_id\":\"ch1\",\"type\":\"system_add_to_channel\",\"message\":\"добавил бота в канал\"}"}}`

	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestHandleWSEvent_IgnoresNonPostedEvent: чужие события не обрабатываем — как и
// раньше, отвечаем 200, не доходя до сервиса.
func TestHandleWSEvent_IgnoresNonPostedEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{service: nil}
	engine := gin.New()
	engine.POST("/event", h.handleWSEvent)

	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(`{"event":"typing","data":{}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
