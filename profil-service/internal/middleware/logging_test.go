package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		if c.GetString("request_id") == "" {
			t.Fatal("request_id absent du contexte")
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "rid-123")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-Id"); got != "rid-123" {
		t.Fatalf("X-Request-Id = %q", got)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	if got := w.Header().Get("X-Request-Id"); len(got) != 32 {
		t.Fatalf("X-Request-Id genere = %q", got)
	}
}

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID(), RequestLogger())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/warn", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
	r.GET("/err", func(c *gin.Context) {
		c.Set("user_id", "u1")
		c.Status(http.StatusInternalServerError)
	})

	for _, path := range []string{"/ok", "/warn", "/err"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path+"?secret=hidden", nil))
	}

	logs := buf.String()
	for _, want := range []string{"level=INFO", "level=WARN", "level=ERROR", "path=/err", "user_id=u1"} {
		if !bytes.Contains([]byte(logs), []byte(want)) {
			t.Fatalf("log sans %q: %s", want, logs)
		}
	}
	if bytes.Contains([]byte(logs), []byte("secret=hidden")) {
		t.Fatalf("la query string ne doit pas etre loguee: %s", logs)
	}
}

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID(), Recovery())
	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d body=%s", w.Code, w.Body.String())
	}
}
