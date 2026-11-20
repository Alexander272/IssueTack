package calendar

import (
	"strconv"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service services.WorkCalendar
}

func NewHandler(service services.WorkCalendar) *Handler {
	return &Handler{service: service}
}

func Register(api *gin.RouterGroup, service services.WorkCalendar, middleware *middleware.Middleware) {
	h := NewHandler(service)
	g := api.Group("/calendar", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		g.GET("", h.get)
	}
}

func (h *Handler) get(c *gin.Context) {
	years := parseYears(c.Query("years"))
	data, err := h.service.Data(c.Request.Context(), years)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func parseYears(raw string) []int {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	years := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		y, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		years = append(years, y)
	}
	return years
}
