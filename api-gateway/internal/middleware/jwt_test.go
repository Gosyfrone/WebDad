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

func makeAdminToken(t *testing.T, secret, role string, ttl time.Duration) string {
	t.Helper()
	return makeToken(t, secret, role, false, ttl)
}

func makeToken(t *testing.T, secret, role string, emailVerified bool, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	cl := &claims{
		UserID:        "11111111-1111-1111-1111-111111111111",
		Email:         "test@breezy.dev",
		Role:          role,
		EmailVerified: emailVerified,
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

// echoHeaders renvoie en JSON les en-têtes d'identité reçus par le handler en
// aval du middleware (donc ceux qui seraient transmis au service backend).
func echoHeaders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"user_id":        c.GetHeader(HeaderUserID),
		"user_role":      c.GetHeader(HeaderUserRole),
		"email_verified": c.GetHeader(HeaderEmailVerified),
	})
}

func TestPropagateJWT_Statuts(t *testing.T) {
	cases := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"token valide", "Bearer " + makeToken(t, testSecret, "user", true, time.Hour), http.StatusOK},
		{"pas de token (passe-plat)", "", http.StatusOK},
		{"token expiré", "Bearer " + makeToken(t, testSecret, "user", true, -time.Hour), http.StatusUnauthorized},
		{"mauvais secret", "Bearer " + makeToken(t, "bad-secret", "user", true, time.Hour), http.StatusUnauthorized},
		{"token bidon", "Bearer not.a.jwt", http.StatusUnauthorized},
		{"format Basic", "Basic abc", http.StatusUnauthorized},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(PropagateJWT(testSecret))
	r.GET("/x", echoHeaders)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
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

func TestPropagateJWT_InjecteLesClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(PropagateJWT(testSecret))
	r.GET("/x", echoHeaders)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, "moderator", true, time.Hour))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `"user_id":"11111111-1111-1111-1111-111111111111"`) {
		t.Fatalf("X-User-Id non propagé : %s", body)
	}
	if !strings.Contains(body, `"user_role":"moderator"`) {
		t.Fatalf("X-User-Role non propagé : %s", body)
	}
	if !strings.Contains(body, `"email_verified":"true"`) {
		t.Fatalf("X-Email-Verified non propagé : %s", body)
	}
}

func TestPropagateJWT_StrippeLesEntetesSpoofes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(PropagateJWT(testSecret))
	r.GET("/x", echoHeaders)

	// Sans token : un client malveillant tente d'usurper son identité via les
	// en-têtes → ils doivent être supprimés avant d'atteindre le handler.
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderUserID, "99999999-9999-9999-9999-999999999999")
	req.Header.Set(HeaderUserRole, "admin")
	req.Header.Set(HeaderEmailVerified, "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `"user_id":""`) || strings.Contains(body, "99999999") {
		t.Fatalf("X-User-Id usurpé non strippé : %s", body)
	}
	if !strings.Contains(body, `"user_role":""`) {
		t.Fatalf("X-User-Role usurpé non strippé : %s", body)
	}
	if !strings.Contains(body, `"email_verified":""`) {
		t.Fatalf("X-Email-Verified usurpé non strippé : %s", body)
	}
}

func TestPropagateJWT_TokenValideEcraseSpoof(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(PropagateJWT(testSecret))
	r.GET("/x", echoHeaders)

	// Token user valide MAIS en-tête de rôle « admin » usurpé : le rôle propagé
	// doit venir des claims (user), pas de l'en-tête client.
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, testSecret, "user", false, time.Hour))
	req.Header.Set(HeaderUserRole, "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `"user_role":"user"`) {
		t.Fatalf("le rôle des claims doit primer sur l'en-tête usurpé : %s", body)
	}
}

// makeNoneToken forge un token signé avec l'algorithme « none » (non-HMAC) :
// vecteur classique d'« alg confusion ». parse doit le rejeter via le garde de
// méthode de signature, sans jamais valider ses claims.
func makeNoneToken(t *testing.T) string {
	t.Helper()
	cl := &claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, cl).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signature token none : %v", err)
	}
	return tok
}

// TestParse_RejetteAlgoNonHMAC couvre le garde anti « alg confusion » : un token
// dont la méthode de signature n'est pas HMAC (ici « none ») est refusé, que ce
// soit via PropagateJWT (passe-plat → 401) ou AdminJWT (401).
func TestParse_RejetteAlgoNonHMAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	none := makeNoneToken(t)

	t.Run("PropagateJWT", func(t *testing.T) {
		r := gin.New()
		r.Use(PropagateJWT(testSecret))
		r.GET("/x", echoHeaders)

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "Bearer "+none)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, attendu 401 (corps: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("AdminJWT", func(t *testing.T) {
		r := gin.New()
		r.GET("/admin", AdminJWT(testSecret), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+none)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, attendu 401 (corps: %s)", w.Code, w.Body.String())
		}
	})
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
