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
		c.JSON(http.StatusOK, gin.H{"ok": true})
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

func TestAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", JWTAuth(testSecret), AdminOnly(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	cases := []struct {
		role       string
		wantStatus int
	}{
		{"admin", http.StatusOK},
		{"user", http.StatusForbidden},
		{"moderator", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, tc.role, time.Hour))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("%s: status = %d, attendu %d", tc.role, w.Code, tc.wantStatus)
			}
		})
	}
}

func TestClaimsFrom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/check", JWTAuth(testSecret), func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, "user", time.Hour))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ClaimsFrom : status = %d", w.Code)
	}
}

func TestParseTokenRejectsNonHMAC(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{})
	raw, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseToken(raw, []byte(testSecret)); err == nil {
		t.Fatal("algorithme non-HMAC accepté")
	}
}
