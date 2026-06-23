package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

// makeToken forge un JWT HS256 signé avec `secret`, expirant dans `ttl`.
func makeToken(t *testing.T, secret string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := Claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "alice@breezy.dev",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signature token : %v", err)
	}
	return signed
}

// newTestRouter monte une route protégée qui renvoie le user_id des claims.
func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", JWTAuth(testSecret), func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no claims"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID})
	})
	return r
}

func TestJWTAuth(t *testing.T) {
	cases := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"token valide", "Bearer " + makeToken(t, testSecret, time.Hour), http.StatusOK},
		{"token expiré", "Bearer " + makeToken(t, testSecret, -time.Hour), http.StatusUnauthorized},
		{"mauvais secret", "Bearer " + makeToken(t, "autre-secret", time.Hour), http.StatusUnauthorized},
		{"token bidon", "Bearer pas.un.jwt", http.StatusUnauthorized},
		{"header absent", "", http.StatusUnauthorized},
		{"format invalide", "Token abc", http.StatusUnauthorized},
	}

	r := newTestRouter()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("%s : status = %d, attendu %d", tc.name, w.Code, tc.wantStatus)
			}
		})
	}
}

// TestClaimsFromAbsent : sans middleware, ClaimsFrom renvoie ok=false.
func TestClaimsFromAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if _, ok := ClaimsFrom(c); ok {
		t.Fatal("ClaimsFrom devrait renvoyer ok=false sans claims posés")
	}
}

// --- OptionalJWTAuth ---

func newOptionalRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/opt", OptionalJWTAuth(testSecret), func(c *gin.Context) {
		_, hasClaims := ClaimsFrom(c)
		if hasClaims {
			c.JSON(http.StatusOK, gin.H{"auth": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"auth": false})
		}
	})
	return r
}

func TestOptionalJWTAuth_SansHeader_Continue(t *testing.T) {
	r := newOptionalRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
}

func TestOptionalJWTAuth_FormatInvalide_Continue(t *testing.T) {
	r := newOptionalRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.Header.Set("Authorization", "Token abc")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200 (format invalide ignoré)", w.Code)
	}
}

func TestOptionalJWTAuth_TokenValide_ClaimsInjectes(t *testing.T) {
	r := newOptionalRouter()
	tok := makeToken(t, testSecret, time.Hour)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
	if w.Body.String() == `{"auth":false}` {
		t.Fatal("claims doivent être injectés avec un token valide")
	}
}

func TestOptionalJWTAuth_TokenInvalide_Continue(t *testing.T) {
	r := newOptionalRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.Header.Set("Authorization", "Bearer pasunevraijwt")
	r.ServeHTTP(w, req)
	// Doit continuer sans erreur (pas de 401)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200 (token invalide ignoré)", w.Code)
	}
}

// --- VerifiedOnly ---

func newVerifiedRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/v", JWTAuth(testSecret), VerifiedOnly(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func makeVerifiedToken(t *testing.T, verified bool) string {
	t.Helper()
	now := time.Now()
	claims := Claims{
		UserID:        "11111111-1111-1111-1111-111111111111",
		Email:         "test@breezy.dev",
		Role:          "user",
		EmailVerified: verified,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeVerifiedToken: %v", err)
	}
	return tok
}

func TestVerifiedOnly_SansClaims_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Route SANS JWTAuth → VerifiedOnly ne trouve pas de claims → 401
	r.GET("/v", VerifiedOnly(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("VerifiedOnly(sans claims) = %d, attendu 401", w.Code)
	}
}

func TestVerifiedOnly_NonVerifie_403(t *testing.T) {
	r := newVerifiedRouter()
	tok := makeVerifiedToken(t, false)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("VerifiedOnly(non vérifié) = %d, attendu 403", w.Code)
	}
}

func TestVerifiedOnly_Verifie_Continue(t *testing.T) {
	r := newVerifiedRouter()
	tok := makeVerifiedToken(t, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("VerifiedOnly(vérifié) = %d, attendu 200", w.Code)
	}
}

// --- ModeratorOnly ---

func makeRoleToken(t *testing.T, role string) string {
	t.Helper()
	now := time.Now()
	claims := Claims{
		UserID:        "11111111-1111-1111-1111-111111111111",
		Email:         "test@breezy.dev",
		Role:          role,
		EmailVerified: true,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeRoleToken: %v", err)
	}
	return tok
}

func newRoleRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/role", JWTAuth(testSecret), mw, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestModeratorOnly_SansClaims_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/role", ModeratorOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/role", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("ModeratorOnly(sans claims) = %d, attendu 401", w.Code)
	}
}

func TestModeratorOnly_RoleUser_403(t *testing.T) {
	r := newRoleRouter(ModeratorOnly())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	req.Header.Set("Authorization", "Bearer "+makeRoleToken(t, "user"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("ModeratorOnly(user) = %d, attendu 403", w.Code)
	}
}

func TestModeratorOnly_RoleModerator_Continue(t *testing.T) {
	r := newRoleRouter(ModeratorOnly())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	req.Header.Set("Authorization", "Bearer "+makeRoleToken(t, "moderator"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ModeratorOnly(moderator) = %d, attendu 200", w.Code)
	}
}

func TestModeratorOnly_RoleAdmin_Continue(t *testing.T) {
	r := newRoleRouter(ModeratorOnly())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	req.Header.Set("Authorization", "Bearer "+makeRoleToken(t, "admin"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ModeratorOnly(admin) = %d, attendu 200", w.Code)
	}
}

// --- AdminOnly ---

func TestAdminOnly_SansClaims_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/role", AdminOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/role", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("AdminOnly(sans claims) = %d, attendu 401", w.Code)
	}
}

func TestAdminOnly_RoleModerator_403(t *testing.T) {
	r := newRoleRouter(AdminOnly())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	req.Header.Set("Authorization", "Bearer "+makeRoleToken(t, "moderator"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("AdminOnly(moderator) = %d, attendu 403", w.Code)
	}
}

func TestAdminOnly_RoleAdmin_Continue(t *testing.T) {
	r := newRoleRouter(AdminOnly())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	req.Header.Set("Authorization", "Bearer "+makeRoleToken(t, "admin"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("AdminOnly(admin) = %d, attendu 200", w.Code)
	}
}

// --- ParseToken ---

func TestParseToken_Valide(t *testing.T) {
	tok := makeToken(t, testSecret, time.Hour)
	claims, err := ParseToken(tok, testSecret)
	if err != nil {
		t.Fatalf("ParseToken valide : %v", err)
	}
	if claims.UserID == "" {
		t.Fatal("ParseToken : UserID attendu dans les claims")
	}
}

func TestParseToken_Invalide(t *testing.T) {
	if _, err := ParseToken("pas.un.jwt", testSecret); err == nil {
		t.Fatal("ParseToken(invalide) doit renvoyer une erreur")
	}
}

func TestParseToken_ExpireToken(t *testing.T) {
	tok := makeToken(t, testSecret, -time.Hour)
	if _, err := ParseToken(tok, testSecret); err == nil {
		t.Fatal("ParseToken(expiré) doit renvoyer une erreur")
	}
}
