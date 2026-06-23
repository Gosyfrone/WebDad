package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"keys":[]}`)) })

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
}
