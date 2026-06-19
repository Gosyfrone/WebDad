package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/user-service/internal/middleware"
	"github.com/webdad/user-service/internal/service"
)

const handlerTestSecret = "handler-test-secret"

// newHandlerRouter monte les handlers avec un UserService à repo nil.
// gin.Recovery() transforme les panics (nil repo) en 500.
func newHandlerRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	svc := service.New(nil, 0)
	h := New(svc)
	auth := middleware.JWTAuth(handlerTestSecret)

	r.GET("/users/search", h.Search)
	r.GET("/users/suggestions", h.Suggestions)
	r.GET("/users/:id", h.GetByID)
	r.GET("/users/by-username/:username", h.GetByUsername)
	r.POST("/users", auth, h.Create)
	r.POST("/users/admin", auth, middleware.AdminOnly(), h.AdminCreate)
	r.GET("/users/me", auth, h.GetMe)
	r.PATCH("/users/me", auth, h.UpdateMe)
	r.PATCH("/users/:id/status", auth, middleware.ModeratorOnly(), h.SetStatus)
	return r
}

func makeHandlerToken(t *testing.T, role string) string {
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
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(handlerTestSecret))
	if err != nil {
		t.Fatalf("makeHandlerToken: %v", err)
	}
	return tok
}

// ─── Search ─────────────────────────────────────────────────────────────────

// UserService.Search("") court-circuite avant d'atteindre le repo nil → 200.
func TestSearch_TermeVide_200(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/search?q=", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /users/search?q= = %d, attendu 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("réponse sans clé 'data' : %s", w.Body.String())
	}
}

func TestSearch_SansParam_200(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /users/search sans ?q = %d, attendu 200", w.Code)
	}
}

// ─── Create — routes protégées ───────────────────────────────────────────────

func TestCreate_SansToken_401(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"username":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /users sans token = %d, attendu 401", w.Code)
	}
}

func TestCreate_AvecToken_JSONInvalide_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST /users JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── AdminCreate — validation username avant repo ────────────────────────────

func TestAdminCreate_SansToken_401(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/users/admin", strings.NewReader(`{"id":"x","username":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

func TestAdminCreate_RoleUser_403(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/users/admin", strings.NewReader(`{"id":"x","username":"alice"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestAdminCreate_JSONInvalide_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeHandlerToken(t, "admin")
	req := httptest.NewRequest(http.MethodPost, "/users/admin", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

// Username trop court → validateUsername → ErrInvalidUsername → 400 (avant repo).
func TestAdminCreate_UsernameTropCourt_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeHandlerToken(t, "admin")
	req := httptest.NewRequest(http.MethodPost, "/users/admin", strings.NewReader(`{"id":"some-id","username":"ab"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("username trop court = %d, attendu 400", w.Code)
	}
}

// ─── GetMe — authentification requise ────────────────────────────────────────

func TestGetMe_SansToken_401(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /users/me sans token = %d, attendu 401", w.Code)
	}
}

// ─── UpdateMe — validation JSON avant service ─────────────────────────────────

func TestUpdateMe_SansToken_401(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/users/me", strings.NewReader(`{"username":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("PATCH /users/me sans token = %d, attendu 401", w.Code)
	}
}

func TestUpdateMe_JSONInvalide_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/users/me", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── SetStatus — modérateur ou admin uniquement ───────────────────────────────

func TestSetStatus_SansToken_401(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/users/some-id/status", strings.NewReader(`{"is_active":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

func TestSetStatus_RoleUser_403(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/users/some-id/status", strings.NewReader(`{"is_active":false}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestSetStatus_JSONInvalide_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeHandlerToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPatch, "/users/some-id/status", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── respondUserError / paginate (chemins couverts via handlers) ──────────────

func TestPaginate_LimitInvalide_UtiliseDéfaut(t *testing.T) {
	r := newHandlerRouter(t)
	// Search court-circuite avant le repo avec q= vide, mais paginate() est
	// appelé au passage avec limit=abc → repli sur defaultLimit → 200.
	req := httptest.NewRequest(http.MethodGet, "/users/search?q=&limit=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("limit invalide = %d, attendu 200", w.Code)
	}
}

func TestPaginate_OffsetNégatif_CorrigéÀZéro(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/search?q=&offset=-5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("offset négatif = %d, attendu 200", w.Code)
	}
}
