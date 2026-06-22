package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ModeratorOnly / AdminOnly montés SANS JWTAuth préalable → claims absents → 401
// (branche `!ok`, distincte du 403 « rôle insuffisant »).

func TestModeratorOnly_SansClaims_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/mod", ModeratorOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/mod", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("ModeratorOnly sans claims = %d, attendu 401", w.Code)
	}
}

func TestAdminOnly_SansClaims_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", AdminOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("AdminOnly sans claims = %d, attendu 401", w.Code)
	}
}

// ParseToken refuse un token signé avec une méthode non-HMAC (ici « none »),
// couvrant le garde-fou de la fonction de validation de clé.
func TestParseToken_MethodeNonHMAC(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{})
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signature none : %v", err)
	}
	if _, err := ParseToken(signed, []byte("k")); err == nil {
		t.Error("ParseToken devrait refuser une signature non-HMAC")
	}
}

// ClaimsFrom sur une valeur de contexte d'un autre type → (nil, false).
func TestClaimsFrom_MauvaisType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(contextKey, "pas-des-claims")
	if _, ok := ClaimsFrom(c); ok {
		t.Error("ClaimsFrom devrait renvoyer ok=false pour un type inattendu")
	}
}
