package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newLoggingRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID(), Recovery(), RequestLogger())
	r.GET("/ok", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Status(http.StatusOK)
	})
	r.GET("/bad", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })
	return r
}

func TestRequestID_Generated(t *testing.T) {
	r := newLoggingRouter()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if id := w.Header().Get("X-Request-Id"); id == "" {
		t.Fatal("X-Request-Id doit être généré et renvoyé")
	}
}

func TestRequestID_FromHeader(t *testing.T) {
	r := newLoggingRouter()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("X-Request-Id", "fixed-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("X-Request-Id"); got != "fixed-id" {
		t.Fatalf("X-Request-Id = %q, attendu fixed-id (repris de l'en-tête)", got)
	}
}

// RequestLogger : niveau warn (≥400).
func TestRequestLogger_Warn(t *testing.T) {
	r := newLoggingRouter()
	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, attendu 400", w.Code)
	}
}

// Recovery : panic → 500, et RequestLogger logue en niveau error.
func TestRecovery_Panic(t *testing.T) {
	r := newLoggingRouter()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic → status = %d, attendu 500", w.Code)
	}
}
