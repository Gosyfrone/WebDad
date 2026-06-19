package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID(), Recovery(), RequestLogger())
	return r
}

func TestRequestID_GénèreSiAbsent(t *testing.T) {
	r := newTestEngine()
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id doit être généré quand absent")
	}
}

func TestRequestID_PropageExistant(t *testing.T) {
	r := newTestEngine()
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-Id", "my-request-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-Id"); got != "my-request-id" {
		t.Fatalf("X-Request-Id propagé = %q, attendu my-request-id", got)
	}
}

func TestRequestLogger_NeBloquesPas(t *testing.T) {
	r := newTestEngine()
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
}

func TestRequestLogger_LogNiveauWarnPour4xx(t *testing.T) {
	r := newTestEngine()
	r.GET("/bad", func(c *gin.Context) { c.Status(http.StatusBadRequest) })

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, attendu 400", w.Code)
	}
}

func TestRequestLogger_LogNiveauErrorPour5xx(t *testing.T) {
	r := newTestEngine()
	r.GET("/err", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, attendu 500", w.Code)
	}
}

func TestRecovery_PanicRetourne500(t *testing.T) {
	r := newTestEngine()
	r.GET("/panic", func(c *gin.Context) { panic("test panic") })

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic → status = %d, attendu 500", w.Code)
	}
}
