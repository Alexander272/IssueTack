package transport

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/ws"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/Alexander272/IssueTrack/backend/pkg/limiter"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/Alexander272/IssueTrack/backend/web"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	keycloak *auth.KeycloakClient
	services *services.Services
	hub      *ws_hub.Hub
}

func NewHandler(keycloak *auth.KeycloakClient, services *services.Services, hub *ws_hub.Hub) *Handler {
	return &Handler{
		keycloak: keycloak,
		services: services,
		hub:      hub,
	}
}

func (h *Handler) Init(conf *config.Config) *gin.Engine {
	router := gin.New()

	// Отключаем редиректы для SPA
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	router.Use(
		gin.LoggerWithConfig(gin.LoggerConfig{
			Skip: func(c *gin.Context) bool {
				path := c.Request.URL.Path
				if strings.HasPrefix(path, "/api") {
					return false
				}
				return c.Writer.Status() < http.StatusBadRequest
			},
		}),
		gin.CustomRecovery(h.ErrorHandler),
		securityHeaders(),
	)

	if err := router.SetTrustedProxies([]string{"192.168.5.0/24", "192.168.4.0/24"}); err != nil {
		logger.Warn("invalid trusted proxies config", logger.ErrAttr(err))
	}

	h.initAPI(router, conf)
	h.initStatic(router, conf)

	return router
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "no-referrer-when-downgrade")
		c.Header("Content-Security-Policy",
			"default-src 'self' http: https: data: blob: 'unsafe-inline'")
		c.Header("Strict-Transport-Security",
			"max-age=31536000; includeSubDomains")
		c.Next()
	}
}

func (h *Handler) ErrorHandler(c *gin.Context, origErr any) {
	err := fmt.Errorf("unexpected error: %v", origErr)

	rawStack := string(debug.Stack())                        // 1. Получаем стек в виде байтов
	cleanStack := strings.ReplaceAll(rawStack, "\t", "    ") // 2. Заменяем все табуляции на 4 пробела для красоты
	stackLines := strings.Split(cleanStack, "\n")            // 3. Превращаем в срез строк, разделяя по символу \n

	// Передаем данные паники в SendError, чтобы избежать дублирования вызова error_bot
	response.SendError(c, err, gin.H{"PANIC": true, "Stack trace": stackLines})
	debug.PrintStack()
}

func (h *Handler) initAPI(router *gin.Engine, conf *config.Config) {
	// middleware := middleware.NewMiddleware(h.services, conf.Auth, h.keycloak)
	// handler := handlers.NewHandler(&handlers.Deps{Services: h.services, Conf: conf, Middleware: middleware})
	// handler := handlers.NewHandler(&handlers.Deps{Services: h.services, Conf: conf, Hub: h.hub})
	wsHandler := ws.NewWsHandler(h.hub, h.services, conf.Http.AllowedOrigins)

	api := router.Group("/api")
	api.Use(limiter.Limit(conf.ApiLimiter.RPS, conf.ApiLimiter.Burst, conf.ApiLimiter.TTL))
	// handler.Init(api)

	api.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	api.GET("/ws", func(c *gin.Context) {
		wsHandler.HandleWS(c)
	})
}

var appStartTime = time.Now()

const (
	frontendRoot = "frontend"
	indexFile    = "index.html"
	assetsPrefix = "assets/"
)

var allowedStaticExts = map[string]bool{
	".html": true, ".js": true, ".css": true, ".png": true, ".jpg": true,
	".jpeg": true, ".svg": true, ".gif": true, ".ico": true, ".webp": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".map": true,
}

func (h *Handler) initStatic(router *gin.Engine, conf *config.Config) {
	router.NoRoute(limiter.Limit(conf.StaticLimiter.RPS, conf.StaticLimiter.Burst, conf.StaticLimiter.TTL), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Status(http.StatusNotFound)
			return
		}

		filePath := strings.TrimPrefix(c.Request.URL.Path, "/")
		if filePath == "" {
			filePath = indexFile
		}
		filePath = path.Clean(filePath)

		// 🔒 Блокируем скрытые файлы/директории (начинаются с точки)
		if strings.HasPrefix(filePath, ".") || strings.Contains(filePath, "/.") {
			c.Status(http.StatusNotFound)
			return
		}

		if ext := path.Ext(filePath); filePath != indexFile && ext != "" && !allowedStaticExts[ext] {
			c.Status(http.StatusNotFound)
			return
		}

		var f fs.File
		var err error
		openPath := frontendRoot + "/" + filePath
		encoding := negotiateEncoding(c.Request.Header.Get("Accept-Encoding"))

		if encoding == "br" {
			f, err = web.Frontend.Open(openPath + ".br")
			if err == nil {
				c.Header("Content-Encoding", "br")
			}
		}
		if f == nil && encoding == "gzip" {
			f, err = web.Frontend.Open(openPath + ".gz")
			if err == nil {
				c.Header("Content-Encoding", "gzip")
			}
		}
		if f == nil {
			f, err = web.Frontend.Open(openPath)
			if err != nil {
				f, err = web.Frontend.Open(frontendRoot + "/" + indexFile)
				if err != nil {
					c.Status(http.StatusNotFound)
					return
				}
				filePath = indexFile
			}
		}
		defer f.Close()

		c.Header("Vary", "Accept-Encoding")

		if strings.HasPrefix(filePath, assetsPrefix) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache")
		}

		if ctype := mime.TypeByExtension(path.Ext(filePath)); ctype != "" {
			c.Header("Content-Type", ctype)
		}

		if rs, ok := f.(io.ReadSeeker); ok {
			http.ServeContent(c.Writer, c.Request, path.Base(filePath), appStartTime, rs)
		} else {
			io.Copy(c.Writer, f)
		}
	})
}

// negotiateEncoding parses Accept-Encoding and returns the best compression
// we can offer: "br", "gzip", or "" (no compression).
func negotiateEncoding(header string) string {
	if header == "" {
		return ""
	}

	type item struct {
		name    string
		quality float64
	}
	var preferred []item
	excluded := map[string]bool{}

	for _, field := range strings.Split(header, ",") {
		enc, q := parseEncodingField(field)
		if enc == "" {
			continue
		}
		if q == 0 {
			if enc == "identity" || enc == "*" {
				return ""
			}
			excluded[enc] = true
			continue
		}
		if enc == "br" || enc == "gzip" || enc == "*" {
			preferred = append(preferred, item{enc, q})
		}
	}

	bestQ := 0.0
	best := ""
	for _, it := range preferred {
		enc := resolveEncoding(it.name, excluded)
		if enc == "" {
			continue
		}
		if it.quality > bestQ || (it.quality == bestQ && enc == "br" && best != "br") {
			best, bestQ = enc, it.quality
		}
	}
	return best
}

func parseEncodingField(field string) (name string, quality float64) {
	field = strings.TrimSpace(field)
	if field == "" {
		return "", 0
	}
	quality = 1.0

	idx := strings.IndexByte(field, ';')
	if idx == -1 {
		return field, quality
	}

	name = strings.TrimSpace(field[:idx])
	rest := field[idx+1:]
	if qi := strings.Index(rest, "q="); qi != -1 {
		qStr := strings.TrimSpace(rest[qi+2:])
		if end := strings.IndexByte(qStr, ';'); end != -1 {
			qStr = strings.TrimSpace(qStr[:end])
		}
		if parsed, err := strconv.ParseFloat(qStr, 64); err == nil {
			quality = parsed
		}
	}
	return name, quality
}

func resolveEncoding(name string, excluded map[string]bool) string {
	if name == "*" {
		if !excluded["gzip"] {
			return "gzip"
		}
		if !excluded["br"] {
			return "br"
		}
		return ""
	}
	if excluded[name] {
		return ""
	}
	return name
}
