package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestClaimsFromMissingAndWrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if claims, ok := ClaimsFrom(c); ok || claims != nil {
		t.Fatalf("ClaimsFrom sans valeur = (%#v, %v)", claims, ok)
	}
	c.Set(contextKey, "not-claims")
	if claims, ok := ClaimsFrom(c); ok || claims != nil {
		t.Fatalf("ClaimsFrom avec mauvais type = (%#v, %v)", claims, ok)
	}
}

func TestParseTokenRejectsUnexpectedSigningMethod(t *testing.T) {
	claims := Claims{
		UserID: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("création du token none: %v", err)
	}
	if _, err := ParseToken(token, []byte("secret")); err == nil {
		t.Fatal("ParseToken doit refuser une méthode de signature non HMAC")
	}
}
