package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ModeratorOnly sans claims → 401 (branche défensive, claims non posés).
func TestModeratorOnly_SansClaims_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/m", ModeratorOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/m", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("ModeratorOnly sans claims = %d, attendu 401", w.Code)
	}
}

// parseToken refuse une méthode de signature non-HMAC (ex. alg=none) : la
// keyfunc renvoie une erreur avant toute validation.
func TestParseToken_RejetteMethodeNonHMAC(t *testing.T) {
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{UserID: "u1"}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("création token none : %v", err)
	}
	if _, err := parseToken(tok, []byte("secret")); err == nil {
		t.Fatal("parseToken doit refuser une signature non-HMAC")
	}
}
