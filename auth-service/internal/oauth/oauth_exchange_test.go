package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

func TestExchangeOAuth2SuccessAndFailures(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("code") == "bad" {
			http.Error(w, "bad code", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","token_type":"Bearer"}`))
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": float64(42), "email": "user@example.test"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	oldClient := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = oldClient }()

	p := &Provider{name: "test", kind: kindOAuth2, oauth: &oauth2.Config{
		ClientID: "id", ClientSecret: "secret", RedirectURL: "https://app.test/callback",
		Endpoint: oauth2.Endpoint{TokenURL: srv.URL + "/token"},
	}, userInfoURL: srv.URL + "/user", userInfoMap: defs["github"].userInfoMap}
	id, err := p.Exchange(context.Background(), "good")
	if err != nil || id.Subject != "42" || id.Email != "user@example.test" {
		t.Fatalf("Exchange = %#v, %v", id, err)
	}
	if _, err := p.Exchange(context.Background(), "bad"); err == nil {
		t.Fatal("bad code accepted")
	}

	p.userInfoURL = srv.URL + "/missing"
	if _, err := p.identityFromUserInfo(context.Background(), &oauth2.Token{AccessToken: "x"}); err == nil {
		t.Fatal("userinfo failure expected")
	}
}

func TestOIDCGuardsAndRegistryDiscovery(t *testing.T) {
	p := &Provider{kind: kindOIDC}
	if _, err := p.identityFromIDToken(context.Background(), &oauth2.Token{}); err == nil {
		t.Fatal("missing id_token accepted")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": server.URL, "authorization_endpoint": server.URL + "/auth",
			"token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		}}})
	})

	old := defs["google"]
	def := old
	def.issuer = server.URL
	defs["google"] = def
	defer func() { defs["google"] = old }()
	reg := NewRegistry(context.Background(), Options{RedirectBaseURL: "https://app.test/", Providers: map[string]Credentials{"google": {ClientID: "id", ClientSecret: "secret"}}})
	provider, err := reg.Get("google")
	if err != nil || provider.verifier == nil || provider.kind != kindOIDC {
		t.Fatalf("OIDC provider = %#v, %v", provider, err)
	}
	badToken := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": "not-a-jwt"})
	if _, err := provider.identityFromIDToken(context.Background(), badToken); err == nil {
		t.Fatal("invalid id_token accepted")
	}

	sign := func(extra map[string]any) string {
		claims := jwt.MapClaims{
			"iss": server.URL, "sub": "subject-42", "aud": "id",
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			"email": "user@example.test",
		}
		for key, value := range extra {
			claims[key] = value
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "test-key"
		raw, signErr := token.SignedString(privateKey)
		if signErr != nil {
			t.Fatal(signErr)
		}
		return raw
	}
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access", "token_type": "Bearer",
			"id_token": sign(map[string]any{"email_verified": true}),
		})
	})

	raw := sign(map[string]any{"email_verified": true})
	identity, err := provider.identityFromIDToken(context.Background(), (&oauth2.Token{}).WithExtra(map[string]any{"id_token": raw}))
	if err != nil || identity.Subject != "subject-42" || identity.Email != "user@example.test" || !identity.EmailVerified {
		t.Fatalf("identity = %#v, %v", identity, err)
	}

	raw = sign(nil)
	identity, err = provider.identityFromIDToken(context.Background(), (&oauth2.Token{}).WithExtra(map[string]any{"id_token": raw}))
	if err != nil || identity.EmailVerified {
		t.Fatalf("missing verified claim = %#v, %v", identity, err)
	}

	raw = sign(map[string]any{"email_verified": "not-a-bool"})
	if _, err := provider.identityFromIDToken(context.Background(), (&oauth2.Token{}).WithExtra(map[string]any{"id_token": raw})); err == nil {
		t.Fatal("malformed email_verified claim accepted")
	}

	identity, err = provider.Exchange(context.Background(), "authorization-code")
	if err != nil || identity.Subject != "subject-42" {
		t.Fatalf("OIDC Exchange = %#v, %v", identity, err)
	}

	defs["unsupported"] = providerDef{kind: providerKind(99)}
	defer delete(defs, "unsupported")
	unsupported := NewRegistry(context.Background(), Options{Providers: map[string]Credentials{"unsupported": {ClientID: "id"}}})
	if _, err := unsupported.Get("unsupported"); err != ErrUnknownProvider {
		t.Fatalf("unsupported kind = %v", err)
	}
}

func TestUserInfoEmailFallbackAndHTTPFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"id":42}`)) })
	mux.HandleFunc("/emails", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusBadGateway) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	oldClient := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = oldClient }()
	p := &Provider{name: "test", userInfoURL: srv.URL + "/user", emailsURL: srv.URL + "/emails", userInfoMap: defs["github"].userInfoMap}
	if _, err := p.identityFromUserInfo(context.Background(), &oauth2.Token{AccessToken: "x"}); err == nil {
		t.Fatal("email fallback failure expected")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out map[string]any
	if err := p.authedGetJSON(ctx, srv.URL+"/user", &oauth2.Token{AccessToken: "x"}, &out); err == nil {
		t.Fatal("cancelled request accepted")
	}
}
