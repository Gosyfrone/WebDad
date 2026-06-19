package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/profil-service/internal/middleware"
	"github.com/webdad/profil-service/internal/service"
)

// newHandlerRouter monte les routes avec un ProfilService à repo nil.
// gin.Recovery() transforme les panics (nil repo) en 500.
func newHandlerRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	// service.New(nil, 0) : service non nil, repo nil — les méthodes qui
	// court-circuitent avant d'atteindre le repo fonctionnent normalement.
	svc := service.New(nil, 0)
	RegisterRoutes(r, "profil-service", svc, "test-secret")
	return r
}

func makeProfilToken(t *testing.T, role string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "user@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("makeProfilToken: %v", err)
	}
	return tok
}

// ─── Search ─────────────────────────────────────────────────────────────────

// GET /profils/search?q= (terme vide) → ProfilService.Search court-circuite
// avant le repo et renvoie []; le handler répond 200 {"data":[]}.
func TestSearch_TermeVide_Retourne200(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/search?q=", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /profils/search?q= = %d, attendu 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("réponse sans clé 'data' : %s", w.Body.String())
	}
}

// GET /profils/search sans ?q → terme vide, même chemin court-circuit.
func TestSearch_SansParam_Retourne200(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /profils/search = %d, attendu 200", w.Code)
	}
}

// searchLimit : ?limit= invalide → repli sur defaultSearchLimit (toujours 200).
func TestSearch_LimitInvalide_UtiliseDéfaut(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/search?q=&limit=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("limit invalide = %d, attendu 200", w.Code)
	}
}

// searchLimit : ?limit=0 → repli sur défaut.
func TestSearch_LimitZero_UtiliseDéfaut(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/search?q=&limit=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("limit=0 = %d, attendu 200", w.Code)
	}
}

// ─── UpdateMe — JSON invalide → 400 avant tout appel service ───────────────

func TestUpdateMe_JSONInvalide_Retourne400(t *testing.T) {
	r := newHandlerRouter(t)
	token := makeProfilToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/profils/me", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PATCH /profils/me JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── Routes protégées sans token → 401 ─────────────────────────────────────

func TestRoutesAuth_SansToken(t *testing.T) {
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/profils"},
		{http.MethodGet, "/profils/me"},
		{http.MethodPatch, "/profils/me"},
		{http.MethodPatch, "/profils/me/activity"},
		{http.MethodPatch, "/profils/me/activity/offline"},
		{http.MethodPost, "/profils/admin"},
	}
	r := newHandlerRouter(t)
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

// ─── Routes publiques accessibles sans token ─────────────────────────────────

func TestRoutesPubliques_SansToken_PasUnauthorized(t *testing.T) {
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/profils/search"},
		{http.MethodGet, "/profils/some-id/visibility"},
		{http.MethodGet, "/profils/some-id/likes-visibility"},
	}
	r := newHandlerRouter(t)
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code == http.StatusUnauthorized {
				t.Fatalf("%s %s public = 401 inattendu", tc.method, tc.path)
			}
		})
	}
}
