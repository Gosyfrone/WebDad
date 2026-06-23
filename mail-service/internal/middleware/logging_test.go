package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ─── RequestID ───────────────────────────────────────────────────────────────

func TestRequestID_GénèreIDSiAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		id := c.GetString("request_id")
		if id == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "id vide"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200 (%s)", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id absent de la réponse")
	}
}

func TestRequestID_PropageIDFourni(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	var gotID string
	r.GET("/", func(c *gin.Context) {
		gotID = c.GetString("request_id")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "my-request-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if gotID != "my-request-id" {
		t.Fatalf("request_id = %q, attendu 'my-request-id'", gotID)
	}
}

// ─── RequestLogger ────────────────────────────────────────────────────────────

func TestRequestLogger_NePasPlanter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID(), RequestLogger())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/err", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })
	r.GET("/bad", func(c *gin.Context) { c.Status(http.StatusBadRequest) })

	for _, path := range []string{"/", "/err", "/bad"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func TestRequestLogger_AvecUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Next()
	})
	r.Use(RequestLogger())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
}

// ─── Recovery ────────────────────────────────────────────────────────────────

func TestRecovery_PanicCapturé_500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) { panic("test panic") })

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic capturé = %d, attendu 500", w.Code)
	}
}
