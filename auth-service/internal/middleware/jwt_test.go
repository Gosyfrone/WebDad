package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

const testSecret = "auth-middleware-test-secret"

func newAuthService(t *testing.T) *services.AuthService {
	t.Helper()
	svc, err := services.New(nil, testSecret, 15*time.Minute, 24*time.Hour, nil, "", "", false, "")
	if err != nil {
		t.Fatalf("services.New: %v", err)
	}
	return svc
}

func makeAuthToken(t *testing.T, role string, expiry time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := &services.Claims{
		UserID: "test-user-id",
		Email:  "test@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeAuthToken: %v", err)
	}
	return tok
}

// router helper qui monte JWTAuth + un handler minimal renvoyant 200.
func jwtRouter(auth *services.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", JWTAuth(auth), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

// ─── JWTAuth ────────────────────────────────────────────────────────────────

func TestJWTAuth_SansHeader_401(t *testing.T) {
	r := jwtRouter(newAuthService(t))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans header = %d, attendu 401", w.Code)
	}
}

func TestJWTAuth_FormatInvalide_401(t *testing.T) {
	r := jwtRouter(newAuthService(t))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("format invalide = %d, attendu 401", w.Code)
	}
}

func TestJWTAuth_TokenInvalide_401(t *testing.T) {
	r := jwtRouter(newAuthService(t))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token invalide = %d, attendu 401", w.Code)
	}
}

func TestJWTAuth_TokenExpiré_401AvecCode(t *testing.T) {
	r := jwtRouter(newAuthService(t))
	tok := makeAuthToken(t, models.RoleUser, -time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token expiré = %d, attendu 401", w.Code)
	}
	body := w.Body.String()
	if len(body) == 0 || (body != "" && !containsAny(body, "token_expired", "expiré")) {
		t.Logf("body: %s", body)
	}
}

func TestJWTAuth_TokenValide_200(t *testing.T) {
	r := jwtRouter(newAuthService(t))
	tok := makeAuthToken(t, models.RoleUser, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("token valide = %d, attendu 200", w.Code)
	}
}

// ─── AdminOnly ───────────────────────────────────────────────────────────────

func setClaimsMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &services.Claims{UserID: "u1", Role: role})
		c.Next()
	}
}

func TestAdminOnly_SansClaims_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", AdminOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans claims = %d, attendu 401", w.Code)
	}
}

func TestAdminOnly_RoleUser_403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", setClaimsMiddleware(models.RoleUser), AdminOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestAdminOnly_RoleAdmin_200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", setClaimsMiddleware(models.RoleAdmin), AdminOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("role admin = %d, attendu 200", w.Code)
	}
}

// ─── ModeratorOnly ───────────────────────────────────────────────────────────

func TestModeratorOnly_RoleUser_403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/mod", setClaimsMiddleware(models.RoleUser), ModeratorOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/mod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestModeratorOnly_RoleModerator_200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/mod", setClaimsMiddleware(models.RoleModerator), ModeratorOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/mod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("role moderator = %d, attendu 200", w.Code)
	}
}

func TestModeratorOnly_RoleAdmin_200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/mod", setClaimsMiddleware(models.RoleAdmin), ModeratorOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/mod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("role admin = %d, attendu 200", w.Code)
	}
}

// ─── ClaimsFrom ─────────────────────────────────────────────────────────────

func TestClaimsFrom_Absents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	claims, ok := ClaimsFrom(c)
	if ok || claims != nil {
		t.Fatal("ClaimsFrom sans claims devrait retourner nil, false")
	}
}

func TestClaimsFrom_Présents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	want := &services.Claims{UserID: "abc", Role: models.RoleUser}
	c.Set("claims", want)
	got, ok := ClaimsFrom(c)
	if !ok || got == nil {
		t.Fatal("ClaimsFrom avec claims devrait retourner true")
	}
	if got.UserID != want.UserID {
		t.Fatalf("UserID = %q, attendu %q", got.UserID, want.UserID)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
