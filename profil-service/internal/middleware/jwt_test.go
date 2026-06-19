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

func TestClaimsFrom_InjecteParJWTAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", JWTAuth(testSecret), func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no claims"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"role": claims.Role, "user_id": claims.UserID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, "admin", time.Hour))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ClaimsFrom inaccessible : status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"role":"admin"`) {
		t.Fatalf("role non transmis : %s", w.Body.String())
	}
}

func TestClaimsFrom_SansContexte(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/no-auth", func(c *gin.Context) {
		_, ok := ClaimsFrom(c)
		if ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "claims présents sans auth"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	req := httptest.NewRequest(http.MethodGet, "/no-auth", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestOptionalJWTAuth_SansToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/pub", OptionalJWTAuth(testSecret), func(c *gin.Context) {
		_, ok := ClaimsFrom(c)
		c.JSON(http.StatusOK, gin.H{"authenticated": ok})
	})

	req := httptest.NewRequest(http.MethodGet, "/pub", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("OptionalJWT sans token = %d, attendu 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"authenticated":false`) {
		t.Fatalf("body = %s, attendu authenticated:false", w.Body.String())
	}
}

func TestOptionalJWTAuth_AvecTokenValide(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/pub", OptionalJWTAuth(testSecret), func(c *gin.Context) {
		_, ok := ClaimsFrom(c)
		c.JSON(http.StatusOK, gin.H{"authenticated": ok})
	})

	req := httptest.NewRequest(http.MethodGet, "/pub", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, "user", time.Hour))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("OptionalJWT avec token = %d, attendu 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"authenticated":true`) {
		t.Fatalf("body = %s, attendu authenticated:true", w.Body.String())
	}
}

func TestOptionalJWTAuth_AvecTokenInvalide(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/pub", OptionalJWTAuth(testSecret), func(c *gin.Context) {
		_, ok := ClaimsFrom(c)
		c.JSON(http.StatusOK, gin.H{"authenticated": ok})
	})

	// Mauvais schéma → pas d'auth mais pas de blocage
	req := httptest.NewRequest(http.MethodGet, "/pub", nil)
	req.Header.Set("Authorization", "Basic invalid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("OptionalJWT schéma invalide = %d, attendu 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"authenticated":false`) {
		t.Fatalf("body = %s, attendu authenticated:false", w.Body.String())
	}
}
