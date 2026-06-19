package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/user-service/internal/middleware"
)

const testSecret = "test-secret"

func makeToken(t *testing.T, role string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "test@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return tok
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// nil UserService — le middleware JWT bloque avant d'atteindre les handlers
	return New(nil, testSecret)
}

func TestHealth(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, attendu 200", w.Code)
	}
}

func TestRoutesPubliques(t *testing.T) {
	// Ces routes ne requièrent pas de JWT.
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/users"},
		{http.MethodGet, "/users/search?q=alice"},
		{http.MethodGet, "/users/suggestions"},
		{http.MethodGet, "/users/by-username/alice"},
		{http.MethodGet, "/users/some-id"},
		{http.MethodGet, "/users/some-id/followers"},
		{http.MethodGet, "/users/some-id/following"},
	}
	r := newTestRouter(t)
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code == http.StatusUnauthorized {
				t.Fatalf("%s %s : route publique ne doit pas exiger de JWT (401)", tc.method, tc.path)
			}
		})
	}
}

func TestRoutesProtégées_SansToken(t *testing.T) {
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/users"},
		{http.MethodGet, "/users/me"},
		{http.MethodPatch, "/users/me"},
		{http.MethodPost, "/users/some-id/follow"},
		{http.MethodDelete, "/users/some-id/follow"},
		{http.MethodGet, "/users/me/follow-requests/outgoing"},
		{http.MethodPost, "/users/follow-requests/abc/accept"},
		{http.MethodPost, "/users/follow-requests/abc/reject"},
		{http.MethodDelete, "/users/me/followers/abc"},
	}
	r := newTestRouter(t)
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s sans token = %d, attendu 401", tc.method, tc.path, w.Code)
			}
		})
	}
}

func TestRouteAdminOnly_SansToken(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/users/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("DELETE /users/:id sans token = %d, attendu 401", w.Code)
	}
}

func TestRouteAdminOnly_TokenUser_Retourne403(t *testing.T) {
	r := newTestRouter(t)
	token := makeToken(t, "user", time.Hour)
	req := httptest.NewRequest(http.MethodDelete, "/users/some-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 403 (pas 401) : le JWT est valide mais le rôle est insuffisant
	if w.Code == http.StatusUnauthorized {
		t.Fatal("token user valide : attendu 403 pas 401")
	}
}

func TestRouteModeratorOnly_TokenUser_Retourne403(t *testing.T) {
	r := newTestRouter(t)
	token := makeToken(t, "user", time.Hour)
	req := httptest.NewRequest(http.MethodPatch, "/users/some-id/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("token user valide sur route mod : attendu 403 pas 401")
	}
}
