package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// TestRequestID_Genere vérifie qu'un id est généré et renvoyé dans l'en-tête.
func TestRequestID_Genere(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		if c.GetString("request_id") == "" {
			t.Error("request_id absent du contexte")
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id absent de la réponse")
	}
}

// TestRequestID_Fourni vérifie que l'id entrant est conservé.
func TestRequestID_Fourni(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "fixe-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-Id") != "fixe-123" {
		t.Errorf("X-Request-Id = %q, attendu 'fixe-123'", w.Header().Get("X-Request-Id"))
	}
}

// TestRequestLogger_TousNiveaux exerce les branches info (200), warn (400) et
// error (500), ainsi que l'ajout de user_id quand présent.
func TestRequestLogger_TousNiveaux(t *testing.T) {
	cases := []struct {
		path   string
		status int
		uid    string
	}{
		{"/ok", http.StatusOK, "u1"},
		{"/bad", http.StatusBadRequest, ""},
		{"/err", http.StatusInternalServerError, ""},
	}
	for _, tc := range cases {
		r := gin.New()
		r.Use(RequestID(), RequestLogger())
		r.GET(tc.path, func(c *gin.Context) {
			if tc.uid != "" {
				c.Set("user_id", tc.uid)
			}
			c.Status(tc.status)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.status {
			t.Errorf("%s → %d, attendu %d", tc.path, w.Code, tc.status)
		}
	}
}

// TestRecovery_Panic500 vérifie qu'une panic est convertie en 500.
func TestRecovery_Panic500(t *testing.T) {
	r := gin.New()
	r.Use(RequestID(), Recovery())
	r.GET("/boom", func(c *gin.Context) { panic("boom") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic → %d, attendu 500", w.Code)
	}
}
