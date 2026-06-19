package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/middleware"
)

func makeToken(t *testing.T, secret, role string, ttl time.Duration) string {
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
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("token : %v", err)
	}
	return tok
}

const testSecret = "test-secret"

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery()) // attrape les panics nil-store dans les tests
	cfg := &config.Config{JWTSecret: testSecret}
	// store nil — les routes protégées renvoient 401 avant d'atteindre le handler
	RegisterRoutes(r, "media-service", nil, cfg)
	return r
}

func TestHealthOK(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, attendu 200", w.Code)
	}
}

func TestRoutesProtégéesSansToken(t *testing.T) {
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/media"},
		{http.MethodPost, "/media/encrypted"},
		{http.MethodDelete, "/media/some-id"},
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

func TestDownloadPublic(t *testing.T) {
	// GET /media/:id est public (pas de JWT requis) — le store nil provoque un 500
	// via gin.Recovery(), mais jamais un 401.
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/media/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("GET /media/:id public ne doit pas retourner 401")
	}
	// 500 attendu (store nil), pas 404 ni 401
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GET /media/:id sans store = %d, attendu 500", w.Code)
	}
}

func TestAdminPurge_SansToken(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/media/owners/some-user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("purge sans token = %d, attendu 401", w.Code)
	}
}

func TestAdminPurge_TokenUser(t *testing.T) {
	r := newTestRouter(t)
	token := makeToken(t, testSecret, "user", time.Hour)
	req := httptest.NewRequest(http.MethodDelete, "/media/owners/some-user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// User ne peut pas purger (403), mais pas 401
	if w.Code == http.StatusUnauthorized {
		t.Fatal("token valide user : attendu 403, pas 401")
	}
}
