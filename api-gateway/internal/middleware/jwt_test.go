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

func makeAdminToken(t *testing.T, secret, role string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	cl := &claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "test@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, cl).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signature token : %v", err)
	}
	return tok
}

func TestAdminJWT(t *testing.T) {
	cases := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"admin valide", "Bearer " + makeAdminToken(t, testSecret, "admin", time.Hour), http.StatusOK},
		{"user refusé", "Bearer " + makeAdminToken(t, testSecret, "user", time.Hour), http.StatusForbidden},
		{"moderator refusé", "Bearer " + makeAdminToken(t, testSecret, "moderator", time.Hour), http.StatusForbidden},
		{"token expiré", "Bearer " + makeAdminToken(t, testSecret, "admin", -time.Hour), http.StatusUnauthorized},
		{"mauvais secret", "Bearer " + makeAdminToken(t, "bad-secret", "admin", time.Hour), http.StatusUnauthorized},
		{"header absent", "", http.StatusUnauthorized},
		{"token bidon", "Bearer not.a.jwt", http.StatusUnauthorized},
		{"schéma Basic", "Basic admin:pass", http.StatusUnauthorized},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin/resource", AdminJWT(testSecret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin/resource", nil)
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

func TestAdminJWT_SecretVide(t *testing.T) {
	// Secret vide → fail-closed : aucun token ne passe
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", AdminJWT(""), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	tok := makeAdminToken(t, testSecret, "admin", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("secret vide doit être fail-closed (refus)")
	}
}
