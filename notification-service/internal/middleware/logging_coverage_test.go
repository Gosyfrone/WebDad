package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDProvidedAndGenerated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		provided string
	}{
		{"fourni", "request-id"},
		{"généré", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(RequestID())
			r.GET("/", func(c *gin.Context) {
				if c.GetString("request_id") == "" {
					t.Error("request_id absent du contexte")
				}
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.provided != "" {
				req.Header.Set("X-Request-Id", tt.provided)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			got := w.Header().Get("X-Request-Id")
			if got == "" {
				t.Fatal("X-Request-Id absent de la réponse")
			}
			if tt.provided != "" && got != tt.provided {
				t.Fatalf("X-Request-Id = %q, attendu %q", got, tt.provided)
			}
			if tt.provided == "" && len(got) != 32 {
				t.Fatalf("X-Request-Id généré = %q, longueur attendue 32", got)
			}
		})
	}
}

func TestRequestLoggerStatusBranches(t *testing.T) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	gin.SetMode(gin.TestMode)
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			r := gin.New()
			r.Use(RequestLogger())
			r.GET("/logged", func(c *gin.Context) {
				c.Set("request_id", "rid")
				c.Set("user_id", "user")
				c.Status(status)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/logged?secret=ignored", nil))
			if w.Code != status {
				t.Fatalf("status = %d, attendu %d", w.Code, status)
			}
		})
	}

	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/anonymous", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/anonymous", nil))
}

func TestRecoveryReturnsInternalServerError(t *testing.T) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID(), Recovery())
	r.GET("/panic", func(*gin.Context) { panic("boom") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, attendu 500", w.Code)
	}
}
