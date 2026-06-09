package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/notification-service/internal/realtime"
)

// newTestRouter monte les routes avec un service nil : on ne teste ici que le
// rejet d'authentification / de secret (les handlers ne sont jamais atteints).
func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, "notification-service", nil, realtime.NewHub(), "test-secret", "internal-secret", []string{"http://localhost:3000"})
	return r
}

func TestProtectedRoutes_RequireAuth(t *testing.T) {
	r := newTestRouter()

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/notifications"},
		{http.MethodGet, "/notifications/unread-count"},
		{http.MethodPost, "/notifications/read"},
		{http.MethodPost, "/notifications/abc/read"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s : attendu 401, obtenu %d", tc.method, tc.path, w.Code)
		}
	}
}

// TestInternalEvents_RequiresSecret : l'ingestion refuse un mauvais secret (401)
// avant d'atteindre le service (nil) — la route est bien protégée.
func TestInternalEvents_RequiresSecret(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/events", strings.NewReader(`{"type":"like"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "wrong")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("/internal/events (mauvais secret) : attendu 401, obtenu %d", w.Code)
	}
}

func TestHealth_OK(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/health : attendu 200, obtenu %d", w.Code)
	}
}

// TestWS_RejectsBadToken : la poignée WS sans token valide répond 401.
func TestWS_RejectsBadToken(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/notifications/ws?access_token=bogus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("/notifications/ws (token invalide) : attendu 401, obtenu %d", w.Code)
	}
}
