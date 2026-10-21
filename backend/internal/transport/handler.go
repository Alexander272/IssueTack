package transport

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/ws"
	"github.com/Alexander272/IssueTrack/backend/pkg/accept_encoding"
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
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self' ws: wss:; "+
				"frame-ancestors 'none'; base-uri 'self'; form-action 'self';")
		c.Header("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), gyroscope=(), "+
				"accelerometer=(), magnetometer=(), usb=(), payment=(), "+
				"display-capture=()")
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
	mw := middleware.NewMiddleware(h.services, &conf.Auth, h.keycloak)
	handler := handlers.NewHandler(&handlers.Deps{Services: h.services, Conf: conf, Middleware: mw})
	wsHandler := ws.NewWsHandler(h.hub, h.services, conf.Http.AllowedOrigins)

	api := router.Group("/api")
	api.Use(limiter.Limit(conf.ApiLimiter.RPS, conf.ApiLimiter.Burst, conf.ApiLimiter.TTL))
	handler.Init(api)

	api.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	api.GET("/ws", mw.VerifyToken, func(c *gin.Context) {
		wsHandler.HandleWS(c)
	})
}

var appStartTime = time.Now()

const (
	frontendRoot = "frontend"
	indexFile    = "index.html"
	assetsPrefix = "assets/"
)

// staticFS — встроенный фронтенд (backend/web/frontend) отдельным FS с корнем в
// этой папке: путь запроса не нужно префиксовать вручную.
var staticFS = mustSub(web.Frontend, frontendRoot)

// mustSub — fs.Sub для константного пути: ошибка означает ошибку сборки.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic("static fs: " + err.Error())
	}
	return sub
}

func (h *Handler) initStatic(router *gin.Engine, conf *config.Config) {
	static := &staticHandler{fsys: staticFS}
	router.NoRoute(limiter.Limit(conf.StaticLimiter.RPS, conf.StaticLimiter.Burst, conf.StaticLimiter.TTL), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Status(http.StatusNotFound)
			return
		}
		static.serve(c)
	})
}

// staticHandler раздаёт встроенный фронтенд: файлы сборки и оболочку SPA для
// маршрутов клиента.
type staticHandler struct {
	fsys fs.FS
}

// serve отдаёт файл статики, а для путей, которых в сборке нет, — оболочку SPA.
func (s *staticHandler) serve(c *gin.Context) {
	filePath := staticPath(c.Request.URL.Path)
	encoding := accept_encoding.Negotiate(c.Request.Header.Get("Accept-Encoding"))

	f, enc, err := openStatic(s.fsys, filePath, encoding)
	if err != nil {
		// Файла нет: путь без расширения — маршрут SPA (отдаём оболочку), с
		// расширением — запрос файла (404: HTML вместо картинки не подсовываем).
		if path.Ext(filePath) != "" {
			c.Status(http.StatusNotFound)
			return
		}
		f, enc, err = openStatic(s.fsys, indexFile, encoding)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		filePath = indexFile
	}
	defer f.Close()

	// Каталог не отдаём: embed отдаёт для него «папку», а читать её нельзя.
	if info, statErr := f.Stat(); statErr == nil && info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Содержимое зависит от Accept-Encoding (сжатый вариант или обычный) — кэш
	// обязан это учитывать.
	c.Header("Vary", "Accept-Encoding")
	if enc != "" {
		c.Header("Content-Encoding", enc)
	}
	if strings.HasPrefix(filePath, assetsPrefix) {
		// Имена ассетов — с хешем: содержимое не меняется, кэшируем надолго.
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}

	http.ServeContent(c.Writer, c.Request, path.Base(filePath), appStartTime, rs)
}

// staticPath превращает путь запроса в относительный путь внутри FS: без
// ведущего слэша, без «..». Дальше путь проверяет сам FS: embed.FS отбрасывает
// невалидные пути (fs.ValidPath), поэтому выйти за пределы сборки нельзя.
func staticPath(urlPath string) string {
	p := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if p == "" || p == "." {
		return indexFile
	}
	return p
}

// openStatic открывает файл статики с учётом Accept-Encoding: если рядом есть
// заранее сжатый вариант (.br/.gz) и клиент его принимает — отдаём его с
// соответствующим Content-Encoding. Сжатые варианты делает сборка образа
// (Dockerfile: gzip -9 + brotli -q 11).
func openStatic(fsys fs.FS, filePath, acceptEncoding string) (fs.File, string, error) {
	if suffix := compressedSuffix(acceptEncoding); suffix != "" {
		if f, err := fsys.Open(filePath + suffix); err == nil {
			return f, acceptEncoding, nil
		}
	}
	f, err := fsys.Open(filePath)
	return f, "", err
}

// compressedSuffix — расширение заранее сжатого варианта файла для кодирования
// («gzip» кодируется файлом «.gz»).
func compressedSuffix(encoding string) string {
	switch encoding {
	case "br":
		return ".br"
	case "gzip":
		return ".gz"
	}
	return ""
}
