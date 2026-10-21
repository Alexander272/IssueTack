package transport

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStaticFS — сборка фронтенда: оболочка SPA, ассет с хешем в имени и
// заранее сжатые варианты (их делает сборка образа).
func testStaticFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                &fstest.MapFile{Data: []byte("<html>shell</html>")},
		"index.html.gz":             &fstest.MapFile{Data: []byte("gz-shell")},
		"assets/app-abc123.js":      &fstest.MapFile{Data: []byte("console.log(1)")},
		"assets/app-abc123.js.gz":   &fstest.MapFile{Data: []byte("gzip-js")},
		"assets/app-abc123.js.br":   &fstest.MapFile{Data: []byte("brotli-js")},
		"assets/logo-abc123.png":    &fstest.MapFile{Data: []byte("png")},
		"assets/vendor-aaa-123.css": &fstest.MapFile{Data: []byte("body{}")},
	}
}

func newStaticRouter(fsys fs.FS) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	static := &staticHandler{fsys: fsys}
	router.NoRoute(static.serve)
	return router
}

func doStatic(router *gin.Engine, path, acceptEncoding string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestStaticHandler_SPARoute: маршрут клиента (без расширения) отдаёт оболочку —
// иначе после перезагрузки страницы был бы 404.
func TestStaticHandler_SPARoute(t *testing.T) {
	router := newStaticRouter(testStaticFS())

	rec := doStatic(router, "/tasks/123", "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "<html>shell</html>", rec.Body.String())
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
}

// TestStaticHandler_Asset: ассет с хешем в имени — иммутабельный, кэшируем на год.
func TestStaticHandler_Asset(t *testing.T) {
	router := newStaticRouter(testStaticFS())

	rec := doStatic(router, "/assets/app-abc123.js", "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "console.log(1)", rec.Body.String())
	assert.Equal(t, "public, max-age=31536000, immutable", rec.Header().Get("Cache-Control"))
	assert.Contains(t, rec.Header().Get("Content-Type"), "javascript")
}

// TestStaticHandler_CompressedVariant: клиент, принимающий сжатие, получает
// заранее сжатый вариант с соответствующим Content-Encoding.
func TestStaticHandler_CompressedVariant(t *testing.T) {
	router := newStaticRouter(testStaticFS())

	t.Run("gzip", func(t *testing.T) {
		rec := doStatic(router, "/assets/app-abc123.js", "gzip")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
		assert.Equal(t, "gzip-js", rec.Body.String())
		// Тип — по исходному имени, а не по сжатому файлу.
		assert.Contains(t, rec.Header().Get("Content-Type"), "javascript")
	})

	t.Run("brotli", func(t *testing.T) {
		rec := doStatic(router, "/assets/app-abc123.js", "br")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "br", rec.Header().Get("Content-Encoding"))
		assert.Equal(t, "brotli-js", rec.Body.String())
	})

	t.Run("оболочка тоже сжата", func(t *testing.T) {
		rec := doStatic(router, "/tasks/123", "gzip")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
		assert.Equal(t, "gz-shell", rec.Body.String())
	})

	// Ответ зависит от Accept-Encoding — кэш обязан это учитывать.
	rec := doStatic(router, "/assets/app-abc123.js", "gzip")
	assert.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))
}

// TestStaticHandler_MissingFile: запрос файла (с расширением), которого нет, —
// 404, а не HTML оболочки.
func TestStaticHandler_MissingFile(t *testing.T) {
	router := newStaticRouter(testStaticFS())

	rec := doStatic(router, "/assets/missing.png", "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NotContains(t, rec.Body.String(), "<html>", "вместо файла нельзя отдавать оболочку")
}

// TestStaticHandler_Directory: каталог не отдаём (раньше получался пустой 200).
func TestStaticHandler_Directory(t *testing.T) {
	router := newStaticRouter(testStaticFS())

	rec := doStatic(router, "/assets/", "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestStaticHandler_NoTraversal: путь за пределы сборки не ведёт — «..» чистится
// на уровне пути, а сама FS отбрасывает невалидные пути (fs.ValidPath).
func TestStaticHandler_NoTraversal(t *testing.T) {
	fsys := fstest.MapFS{
		"public/index.html": &fstest.MapFile{Data: []byte("<html>shell</html>")},
		"secret.txt":        &fstest.MapFile{Data: []byte("secret")},
	}
	public, err := fs.Sub(fsys, "public")
	require.NoError(t, err)
	router := newStaticRouter(public)

	for _, p := range []string{"/../secret.txt", "/assets/../../secret.txt", "//../secret.txt"} {
		rec := doStatic(router, p, "")
		assert.NotContains(t, rec.Body.String(), "secret", "путь %s не должен отдавать файл", p)
	}
}

// TestStaticHandler_ConditionalRequest: у встроенных файлов ModTime нулевой,
// поэтому стабильный Last-Modified даёт 304 вместо полной отдачи.
func TestStaticHandler_ConditionalRequest(t *testing.T) {
	router := newStaticRouter(testStaticFS())

	rec := doStatic(router, "/index.html", "")
	require.Equal(t, http.StatusOK, rec.Code)
	lastModified := rec.Header().Get("Last-Modified")
	require.NotEmpty(t, lastModified)

	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	req.Header.Set("If-Modified-Since", lastModified)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusNotModified, rec2.Code)
}
