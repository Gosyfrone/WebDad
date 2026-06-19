package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/report-service/internal/middleware"
	"github.com/webdad/report-service/internal/service"
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
	// Passer nil comme service — les routes protégées renvoient 401 avant d'atteindre le handler
	RegisterRoutes(r, "report-service", (*service.ReportService)(nil), testSecret)
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
		{http.MethodPost, "/reports"},
		{http.MethodGet, "/reports/warnings/pending"},
		{http.MethodGet, "/reports/tickets"},
		{http.MethodGet, "/reports/settings"},
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

func TestModRoutesSansRoleMod(t *testing.T) {
	r := newTestRouter(t)
	// Avec un token user, les routes modérateur doivent retourner 403
	token := makeToken(t, testSecret, "user", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/reports/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("avec token user valide, attendu 403 (pas 401)")
	}
}

func TestAdminRouteSansRoleAdmin(t *testing.T) {
	r := newTestRouter(t)
	token := makeToken(t, testSecret, "moderator", time.Hour)
	req := httptest.NewRequest(http.MethodPatch, "/reports/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("avec token moderator, attendu 403 (pas 401)")
	}
}
