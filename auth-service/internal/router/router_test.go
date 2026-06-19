package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// nil auth/oauthReg — le middleware JWT bloque avant d'atteindre les handlers
	// pour les routes protégées ; les routes publiques sont accessibles.
	return New(nil, nil)
}

func TestHealth(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, attendu 200", w.Code)
	}
}

func TestRoutesPubliques_AcceptéesSansToken(t *testing.T) {
	// Ces routes ne portent pas de middleware JWT — elles ne doivent pas rejeter
	// un appel sans token par un 401 (elles peuvent échouer autrement car le
	// service est nil, mais la garde JWT n'intervient pas).
	r := newTestRouter(t)
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/auth/register"},
		{http.MethodPost, "/auth/login"},
		{http.MethodPost, "/auth/verify-email/confirm"},
		{http.MethodPost, "/auth/verify-email/request"},
		{http.MethodPost, "/auth/password/forgot"},
		{http.MethodPost, "/auth/password/reset"},
		{http.MethodPost, "/auth/logout"},
		{http.MethodPost, "/auth/mfa/verify"},
		{http.MethodPost, "/auth/email/change/confirm"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code == http.StatusUnauthorized {
				t.Fatalf("%s %s : route publique ne doit pas exiger de JWT (401)", tc.method, tc.path)
			}
		})
	}
}

func TestRoutesProtégées_SansToken_Rejettent401(t *testing.T) {
	r := newTestRouter(t)
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/auth/password/change"},
		{http.MethodPost, "/auth/email/change/request"},
		{http.MethodPost, "/auth/mfa/setup"},
		{http.MethodPost, "/auth/mfa/enable"},
		{http.MethodPost, "/auth/mfa/disable"},
		{http.MethodGet, "/auth/mfa/status"},
		{http.MethodGet, "/auth/validate"},
		{http.MethodGet, "/auth/users"},
		{http.MethodPost, "/auth/users"},
		{http.MethodPatch, "/auth/users/some-id/status"},
		{http.MethodPatch, "/auth/users/some-id/role"},
		{http.MethodDelete, "/auth/users/some-id"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s sans token = %d, attendu 401", tc.method, tc.path, w.Code)
			}
		})
	}
}

func TestOAuthRoutes_AcceptéesSansToken(t *testing.T) {
	r := newTestRouter(t)
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/auth/oauth/google/url"},
		{http.MethodPost, "/auth/oauth/google/exchange"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code == http.StatusUnauthorized {
				t.Fatalf("%s %s : route OAuth ne doit pas exiger de JWT (401)", tc.method, tc.path)
			}
		})
	}
}
