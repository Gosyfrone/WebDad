package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCORSRouter(allowed []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(allowed))
	r.GET("/test", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.POST("/test", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func TestCORS_Preflight_OrigineAutorisée(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, attendu 204", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("ACAO header absent ou incorrect : %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("Access-Control-Allow-Methods absent")
	}
}

func TestCORS_GET_OrigineAutorisée(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, attendu 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("ACAO header absent")
	}
}

func TestCORS_OrigineNonAutorisée(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("origine non autorisée ne doit pas recevoir le header CORS")
	}
}

func TestCORS_SansOrigin(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("sans Origin status = %d, attendu 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("sans Origin ne doit pas poser le header CORS")
	}
}

func TestCORS_MultiOrigines(t *testing.T) {
	r := newCORSRouter([]string{"http://app.breezy.dev", "http://localhost:3000"})

	for _, origin := range []string{"http://app.breezy.dev", "http://localhost:3000"} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Fatalf("origine %q non reconnue", origin)
		}
	}
}
