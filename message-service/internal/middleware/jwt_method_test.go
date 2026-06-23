package middleware

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// ParseToken doit rejeter un token signé avec une méthode NON-HMAC (la keyfunc
// n'accepte que HS*) → couvre la garde sur la méthode de signature.
func TestParseToken_NonHMACMethod(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u"})
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signature 'none' : %v", err)
	}
	if _, err := ParseToken(signed, []byte("secret")); err == nil {
		t.Fatal("un token non-HMAC doit être rejeté")
	}
}
