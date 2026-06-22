package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// newLoggingRouter monte les trois middlewares transverses + une route cible.
func newLoggingRouter(handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(RequestID(), RequestLogger(), Recovery())
	r.GET("/ok", handler)
	return r
}

func TestRequestID_GeneratedAndEchoed(t *testing.T) {
	r := newLoggingRouter(func(c *gin.Context) {
		if c.GetString("request_id") == "" {
			t.Error("request_id doit être posé dans le contexte")
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if w.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id doit être renvoyé")
	}
}

func TestRequestID_PreservesIncoming(t *testing.T) {
	r := newLoggingRouter(func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("X-Request-Id", "fixed-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-Id") != "fixed-id" {
		t.Errorf("X-Request-Id = %q, attendu fixed-id", w.Header().Get("X-Request-Id"))
	}
}

// RequestLogger emprunte chaque branche de niveau (info/warn/error) selon le statut.
func TestRequestLogger_StatusLevels(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError} {
		r := gin.New()
		r.Use(RequestID(), RequestLogger())
		r.GET("/ok", func(c *gin.Context) {
			c.Set("user_id", "u1")
			c.Status(status)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))
		if w.Code != status {
			t.Errorf("statut = %d, attendu %d", w.Code, status)
		}
	}
}

// Recovery transforme un panic en 500.
func TestRecovery_PanicTo500(t *testing.T) {
	r := newLoggingRouter(func(c *gin.Context) { panic("boom") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic → %d, attendu 500", w.Code)
	}
}
