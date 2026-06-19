package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func makeToken(t *testing.T, secret, role string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := Claims{
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
		t.Fatalf("signature token : %v", err)
	}
	return tok
}

func TestJWTAuth(t *testing.T) {
	cases := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"token valide", "Bearer " + makeToken(t, testSecret, "user", time.Hour), http.StatusOK},
		{"token expiré", "Bearer " + makeToken(t, testSecret, "user", -time.Hour), http.StatusUnauthorized},
		{"mauvais secret", "Bearer " + makeToken(t, "autre-secret", "user", time.Hour), http.StatusUnauthorized},
		{"token bidon", "Bearer pas.un.jwt", http.StatusUnauthorized},
		{"header absent", "", http.StatusUnauthorized},
		{"schéma manquant", makeToken(t, testSecret, "user", time.Hour), http.StatusUnauthorized},
		{"mauvais schéma", "Basic xyz", http.StatusUnauthorized},
	}

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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, attendu %d (corps: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	key := []byte(testSecret)
	tok := makeToken(t, testSecret, "user", time.Hour)
	claims, err := ParseToken(tok, key)
	if err != nil {
		t.Fatalf("ParseToken valid = %v", err)
	}
	if claims.Role != "user" {
		t.Fatalf("role = %q, attendu user", claims.Role)
	}

	// Token expiré
	expired := makeToken(t, testSecret, "user", -time.Hour)
	if _, err := ParseToken(expired, key); err == nil {
		t.Fatal("ParseToken expiré doit retourner une erreur")
	}
}

func TestAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", JWTAuth(testSecret), AdminOnly(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	cases := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{"admin", "admin", http.StatusOK},
		{"user", "user", http.StatusForbidden},
		{"moderator", "moderator", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, tc.role, time.Hour))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("%s: status = %d, attendu %d", tc.name, w.Code, tc.wantStatus)
			}
		})
	}
}

func TestModeratorOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/mod", JWTAuth(testSecret), ModeratorOnly(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	cases := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{"admin accepté", "admin", http.StatusOK},
		{"moderator accepté", "moderator", http.StatusOK},
		{"user refusé", "user", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/mod", nil)
			req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, tc.role, time.Hour))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("%s: status = %d, attendu %d", tc.name, w.Code, tc.wantStatus)
			}
		})
	}
}

func TestClaimsFrom_SansContexte(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		_, ok := ClaimsFrom(c)
		c.JSON(http.StatusOK, gin.H{"found": ok})
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), `"found":false`) {
		t.Fatalf("attendu found:false, got %s", w.Body.String())
	}
}
