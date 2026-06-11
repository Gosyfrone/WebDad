package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/realtime"
)

// newTestRouter monte les routes avec un service nil : on ne teste ici que le
// rejet d'authentification (le handler n'est jamais atteint sans token valide).
func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, "message-service", nil, realtime.NewHub(), "test-secret", []string{"http://localhost:3000"})
	return r
}

func TestProtectedRoutes_RequireAuth(t *testing.T) {
	r := newTestRouter()

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/messages/conversations"},
		{http.MethodPost, "/messages/conversations"},
		{http.MethodGet, "/messages/conversations/abc"},
		{http.MethodGet, "/messages/conversations/abc/messages"},
		{http.MethodPost, "/messages/conversations/abc/messages"},
		{http.MethodPatch, "/messages/conversations/abc/messages/def"},
		{http.MethodPatch, "/messages/conversations/abc"},
		{http.MethodDelete, "/messages/conversations/abc"},
		{http.MethodGet, "/messages/conversations/abc/members"},
		{http.MethodPost, "/messages/conversations/abc/members"},
		{http.MethodPatch, "/messages/conversations/abc/members/u2"},
		{http.MethodDelete, "/messages/conversations/abc/members/u2"},
		{http.MethodPost, "/messages/conversations/abc/join"},
		{http.MethodPatch, "/messages/conversations/abc/pin"},
		{http.MethodDelete, "/messages/conversations/abc/pin"},
		{http.MethodDelete, "/messages/conversations/abc/me"},
		{http.MethodGet, "/messages/communities"},
		{http.MethodPut, "/messages/keys"},
		{http.MethodGet, "/messages/keys/u1"},
		{http.MethodPut, "/messages/keys/backup"},
		{http.MethodGet, "/messages/keys/backup"},
		{http.MethodGet, "/messages/keys/backup/status"},
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

func TestHealth_OK(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/health : attendu 200, obtenu %d", w.Code)
	}
}

// TestWS_RejectsBadToken : la poignée WS sans token valide répond 401 (et ne
// tente pas l'upgrade).
func TestWS_RejectsBadToken(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/messages/ws?access_token=bogus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("/messages/ws (token invalide) : attendu 401, obtenu %d", w.Code)
	}
}
