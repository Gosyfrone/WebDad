package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// callGuard exécute une garde de rôle sur un contexte où les claims donnés sont
// (ou non) injectés, et renvoie le statut HTTP résultant.
func callGuard(guard gin.HandlerFunc, claims *Claims) int {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if claims != nil {
		c.Set(contextKey, claims)
	}
	guard(c)
	if !c.IsAborted() {
		return http.StatusOK
	}
	return w.Code
}

func TestVerifiedOnly(t *testing.T) {
	if got := callGuard(VerifiedOnly(), &Claims{EmailVerified: true}); got != http.StatusOK {
		t.Errorf("e-mail vérifié = %d, attendu 200", got)
	}
	if got := callGuard(VerifiedOnly(), &Claims{EmailVerified: false}); got != http.StatusForbidden {
		t.Errorf("e-mail non vérifié = %d, attendu 403", got)
	}
	if got := callGuard(VerifiedOnly(), nil); got != http.StatusUnauthorized {
		t.Errorf("claims absents = %d, attendu 401", got)
	}
}

func TestRoleGuards_ClaimsAbsent(t *testing.T) {
	if got := callGuard(AdminOnly(), nil); got != http.StatusUnauthorized {
		t.Errorf("AdminOnly sans claims = %d, attendu 401", got)
	}
	if got := callGuard(ModeratorOnly(), nil); got != http.StatusUnauthorized {
		t.Errorf("ModeratorOnly sans claims = %d, attendu 401", got)
	}
}
