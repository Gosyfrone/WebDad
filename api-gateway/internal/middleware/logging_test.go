package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_Génère(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		id := c.GetString("request_id")
		if id == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "id vide"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"request_id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d : %s", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id absent dans la réponse")
	}
}

func TestRequestID_Propage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.GetString("request_id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", "my-trace-id-42")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-Id") != "my-trace-id-42" {
		t.Fatalf("X-Request-Id propagé = %q, attendu my-trace-id-42", w.Header().Get("X-Request-Id"))
	}
}

func TestRequestLogger_Passe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("RequestLogger bloque la requête : %d", w.Code)
	}
}

func TestRecovery_CatchPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Recovery : status = %d, attendu 500", w.Code)
	}
	if !strings.Contains(w.Body.String(), "erreur interne") {
		t.Fatalf("body = %s, attendu 'erreur interne'", w.Body.String())
	}
}

func TestUpstreamFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/auth/register", "auth"},
		{"/users/me", "users"},
		{"/posts", "posts"},
		{"/", ""},
		{"", ""},
		{"/profils/123/activity", "profils"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			if got := upstreamFromPath(tc.path); got != tc.want {
				t.Fatalf("upstreamFromPath(%q) = %q, attendu %q", tc.path, got, tc.want)
			}
		})
	}
}
