package error_bot

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

type Message struct {
	Service *Service     `json:"service" binding:"required"`
	Data    *MessageData `json:"data" binding:"required"`
}

type Service struct {
	Id   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type MessageData struct {
	Date    string `json:"date" binding:"required"`
	Error   string `json:"error" binding:"required"`
	IP      string `json:"ip" binding:"required"`
	URL     string `json:"url" binding:"required"`
	Request string `json:"request"`
}

func Send(c *gin.Context, e string, request interface{}) {
	var req []byte
	if request != nil {
		var err error
		req, err = json.MarshalIndent(request, "", "    ")
		if err != nil {
			slog.Error("failed to marshal request body.", slog.String("error", err.Error()))
		}
	}

	data := &MessageData{
		Date:    time.Now().Format("02/01/2006 - 15:04:05"),
		Error:   e,
		Request: string(req),
	}
	if c != nil {
		data.IP = c.ClientIP()
		data.URL = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.String())
	}

	message := Message{
		Service: &Service{
			Id:   "rings",
			Name: "Ring Orders",
		},
		Data: data,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(message); err != nil {
		slog.Error("failed to encode struct.", slog.String("error", err.Error()))
	}

	url := os.Getenv("ERR_URL")
	if url == "" {
		return
	}

	_, err := http.Post(url, "application/json", &buf)
	if err != nil {
		slog.Error("failed to send error to bot.", slog.String("error", err.Error()))
	}
}
