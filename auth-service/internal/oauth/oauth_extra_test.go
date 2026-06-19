// Tests supplémentaires du package oauth.
// Stratégie : pas de dépendance réseau. On teste :
//   - strField / idField (fonctions pures) ;
//   - NewRegistry avec plusieurs configurations ;
//   - Registry.Get() avec des providers configurés mais sans réseau (OAuth2
//     kindOAuth2 ne fait pas de discovery — construit immédiatement) ;
//   - Provider.AuthURL (ne fait pas d'appel réseau) ;
//   - authedGetJSON avec un httptest.Server (mock HTTP) ;
//   - identityFromUserInfo avec mock HTTP ;
//   - fetchPrimaryEmail avec mock HTTP.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// ─── strField ─────────────────────────────────────────────────────────────────

func TestStrField(t *testing.T) {
	m := map[string]any{
		"email":  "user@test.dev",
		"name":   "Alice",
		"number": 42.0,
		"nil":    nil,
	}
	if got := strField(m, "email"); got != "user@test.dev" {
		t.Errorf("strField email = %q, attendu %q", got, "user@test.dev")
	}
	if got := strField(m, "number"); got != "" {
		t.Errorf("strField number = %q, attendu ''", got)
	}
	if got := strField(m, "absent"); got != "" {
		t.Errorf("strField absent = %q, attendu ''", got)
	}
	if got := strField(m, "nil"); got != "" {
		t.Errorf("strField nil = %q, attendu ''", got)
	}
}

// ─── idField ──────────────────────────────────────────────────────────────────

func TestIdField(t *testing.T) {
	m := map[string]any{
		"id_str":   "abc123",
		"id_num":   float64(42),
		"id_other": true,
	}
	if got := idField(m, "id_str"); got != "abc123" {
		t.Errorf("idField string = %q, attendu %q", got, "abc123")
	}
	if got := idField(m, "id_num"); got != "42" {
		t.Errorf("idField float64 = %q, attendu %q", got, "42")
	}
	if got := idField(m, "id_other"); got != "" {
		t.Errorf("idField bool = %q, attendu ''", got)
	}
	if got := idField(m, "absent"); got != "" {
		t.Errorf("idField absent = %q, attendu ''", got)
	}
}

// ─── NewState ─────────────────────────────────────────────────────────────────

func TestNewState_Longueur(t *testing.T) {
	s, err := NewState()
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	// 32 octets en base64url raw = 43 chars.
	if len(s) < 40 {
		t.Errorf("NewState trop court : %d chars", len(s))
	}
}

func TestNewState_NullBytes(t *testing.T) {
	// La valeur ne doit pas contenir de bytes nuls.
	for i := 0; i < 10; i++ {
		s, _ := NewState()
		if strings.Contains(s, "\x00") {
			t.Fatal("NewState contient des bytes nuls")
		}
	}
}

// ─── NewRegistry ──────────────────────────────────────────────────────────────

func TestNewRegistry_PlusieursProviders(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{
		RedirectBaseURL: "http://localhost:3000",
		Providers: map[string]Credentials{
			"google":   {ClientID: ""},         // ignoré
			"github":   {ClientID: "gh-id", ClientSecret: "gh-secret"},
			"facebook": {ClientID: "fb-id", ClientSecret: "fb-secret"},
		},
	})

	// google ignoré.
	if _, err := reg.Get("google"); err != ErrUnknownProvider {
		t.Errorf("google (vide) = %v, attendu ErrUnknownProvider", err)
	}
	// github configuré (kindOAuth2, pas de discovery réseau).
	p, err := reg.Get("github")
	if err != nil {
		t.Fatalf("github configuré = %v", err)
	}
	if p == nil {
		t.Fatal("provider github nil")
	}
	// Cache : deuxième appel retourne le même objet.
	p2, err := reg.Get("github")
	if err != nil {
		t.Fatalf("github (cache) = %v", err)
	}
	if p != p2 {
		t.Fatal("le provider en cache devrait être le même pointeur")
	}
}

func TestNewRegistry_ProviderInconnu(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{
		Providers: map[string]Credentials{
			"unknown-provider": {ClientID: "id", ClientSecret: "secret"},
		},
	})
	// Configuré mais absent de defs → ErrUnknownProvider.
	if _, err := reg.Get("unknown-provider"); err != ErrUnknownProvider {
		t.Errorf("provider absent de defs = %v, attendu ErrUnknownProvider", err)
	}
}

func TestNewRegistry_SpotifyConfiguré(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{
		RedirectBaseURL: "https://app.breezy.dev",
		Providers: map[string]Credentials{
			"spotify": {ClientID: "sp-id", ClientSecret: "sp-secret"},
		},
	})
	p, err := reg.Get("spotify")
	if err != nil {
		t.Fatalf("spotify configuré = %v", err)
	}
	if p == nil {
		t.Fatal("provider spotify nil")
	}
}

func TestNewRegistry_FacebookConfiguré(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{
		RedirectBaseURL: "https://app.breezy.dev",
		Providers: map[string]Credentials{
			"facebook": {ClientID: "fb-id", ClientSecret: "fb-secret"},
		},
	})
	p, err := reg.Get("facebook")
	if err != nil {
		t.Fatalf("facebook configuré = %v", err)
	}
	if p.userInfoURL == "" {
		t.Fatal("userInfoURL vide pour facebook")
	}
}

// ─── Provider.AuthURL ─────────────────────────────────────────────────────────

func TestProvider_AuthURL(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{
		RedirectBaseURL: "http://localhost:3000",
		Providers: map[string]Credentials{
			"github": {ClientID: "my-client-id", ClientSecret: "secret"},
		},
	})
	p, err := reg.Get("github")
	if err != nil {
		t.Fatalf("Get github: %v", err)
	}
	state := "anti-csrf-state"
	url := p.AuthURL(state)
	if url == "" {
		t.Fatal("AuthURL vide")
	}
	if !strings.Contains(url, "github.com") {
		t.Errorf("AuthURL sans github.com: %q", url)
	}
	if !strings.Contains(url, state) {
		t.Errorf("AuthURL sans state: %q", url)
	}
	if !strings.Contains(url, "my-client-id") {
		t.Errorf("AuthURL sans client_id: %q", url)
	}
}

func TestProvider_AuthURL_SpotifyContientScopes(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{
		RedirectBaseURL: "http://localhost:3000",
		Providers: map[string]Credentials{
			"spotify": {ClientID: "sp-id", ClientSecret: "sp-secret"},
		},
	})
	p, _ := reg.Get("spotify")
	url := p.AuthURL("test-state")
	if !strings.Contains(url, "spotify.com") {
		t.Errorf("URL Spotify sans spotify.com: %q", url)
	}
}

// ─── authedGetJSON — avec mock HTTP ──────────────────────────────────────────

func TestAuthedGetJSON_OK(t *testing.T) {
	payload := map[string]any{"id": "user123", "email": "u@test.dev"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vérifie le header Authorization.
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	p := &Provider{name: "test", userInfoURL: srv.URL, userInfoMap: func(m map[string]any) Identity {
		return Identity{Subject: strField(m, "id"), Email: strField(m, "email"), EmailVerified: true}
	}}

	// Fake token (juste un Bearer pour le mock).
	tok := &fakeOAuthToken{access: "test-access-token"}
	var out map[string]any
	if err := p.authedGetJSON(context.Background(), srv.URL, tok.toOAuth2(), &out); err != nil {
		t.Fatalf("authedGetJSON OK: %v", err)
	}
	if out["id"] != "user123" {
		t.Errorf("id = %v, attendu 'user123'", out["id"])
	}
}

func TestAuthedGetJSON_Statut404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p := &Provider{name: "test"}
	tok := &fakeOAuthToken{access: "tok"}
	var out map[string]any
	if err := p.authedGetJSON(context.Background(), srv.URL, tok.toOAuth2(), &out); err == nil {
		t.Fatal("statut 404 devrait retourner une erreur")
	}
}

func TestAuthedGetJSON_JSONInvalide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "not json {{{")
	}))
	defer srv.Close()

	p := &Provider{name: "test"}
	tok := &fakeOAuthToken{access: "tok"}
	var out map[string]any
	if err := p.authedGetJSON(context.Background(), srv.URL, tok.toOAuth2(), &out); err == nil {
		t.Fatal("JSON invalide devrait retourner une erreur")
	}
}

func TestAuthedGetJSON_URLInvalide(t *testing.T) {
	p := &Provider{name: "test"}
	tok := &fakeOAuthToken{access: "tok"}
	var out map[string]any
	if err := p.authedGetJSON(context.Background(), "://not-a-url", tok.toOAuth2(), &out); err == nil {
		t.Fatal("URL invalide devrait retourner une erreur")
	}
}

// ─── fetchPrimaryEmail — avec mock HTTP ───────────────────────────────────────

func TestFetchPrimaryEmail_EmailPrimaire(t *testing.T) {
	emails := []map[string]any{
		{"email": "secondary@test.dev", "primary": false, "verified": true},
		{"email": "primary@test.dev", "primary": true, "verified": true},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(emails)
	}))
	defer srv.Close()

	p := &Provider{name: "github", emailsURL: srv.URL}
	tok := &fakeOAuthToken{access: "tok"}
	email, err := p.fetchPrimaryEmail(context.Background(), tok.toOAuth2())
	if err != nil {
		t.Fatalf("fetchPrimaryEmail: %v", err)
	}
	if email != "primary@test.dev" {
		t.Errorf("email = %q, attendu 'primary@test.dev'", email)
	}
}

func TestFetchPrimaryEmail_SeulementSecondaire(t *testing.T) {
	emails := []map[string]any{
		{"email": "secondary@test.dev", "primary": false, "verified": true},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(emails)
	}))
	defer srv.Close()

	p := &Provider{name: "github", emailsURL: srv.URL}
	tok := &fakeOAuthToken{access: "tok"}
	// Pas de primaire mais un vérifié → repli sur le premier vérifié.
	email, err := p.fetchPrimaryEmail(context.Background(), tok.toOAuth2())
	if err != nil {
		t.Fatalf("fetchPrimaryEmail (secondaire): %v", err)
	}
	if email != "secondary@test.dev" {
		t.Errorf("email = %q, attendu 'secondary@test.dev'", email)
	}
}

func TestFetchPrimaryEmail_AucunVérifié(t *testing.T) {
	emails := []map[string]any{
		{"email": "unverified@test.dev", "primary": true, "verified": false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(emails)
	}))
	defer srv.Close()

	p := &Provider{name: "github", emailsURL: srv.URL}
	tok := &fakeOAuthToken{access: "tok"}
	if _, err := p.fetchPrimaryEmail(context.Background(), tok.toOAuth2()); err == nil {
		t.Fatal("aucun email vérifié devrait retourner une erreur")
	}
}

func TestFetchPrimaryEmail_ListeVide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, "[]")
	}))
	defer srv.Close()

	p := &Provider{name: "github", emailsURL: srv.URL}
	tok := &fakeOAuthToken{access: "tok"}
	if _, err := p.fetchPrimaryEmail(context.Background(), tok.toOAuth2()); err == nil {
		t.Fatal("liste vide devrait retourner une erreur")
	}
}

// ─── identityFromUserInfo — avec mock HTTP ────────────────────────────────────

func TestIdentityFromUserInfo_OK(t *testing.T) {
	userInfo := map[string]any{"id": "github-user-42", "email": "user@github.dev"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userInfo)
	}))
	defer srv.Close()

	p := &Provider{
		name:        "github",
		userInfoURL: srv.URL,
		userInfoMap: defs["github"].userInfoMap,
	}

	// On patche httpClient pour qu'il pointe sur notre mock.
	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	tok := &fakeOAuthToken{access: "test-tok"}
	id, err := p.identityFromUserInfo(context.Background(), tok.toOAuth2())
	if err != nil {
		t.Fatalf("identityFromUserInfo: %v", err)
	}
	if id.Subject == "" {
		t.Error("Subject vide")
	}
	if id.Email != "user@github.dev" {
		t.Errorf("Email = %q, attendu 'user@github.dev'", id.Email)
	}
}

func TestIdentityFromUserInfo_SansSubject(t *testing.T) {
	userInfo := map[string]any{"email": "user@github.dev"} // pas d'id
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userInfo)
	}))
	defer srv.Close()

	p := &Provider{
		name:        "github",
		userInfoURL: srv.URL,
		userInfoMap: func(m map[string]any) Identity {
			return Identity{Subject: "", Email: strField(m, "email")}
		},
	}
	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	tok := &fakeOAuthToken{access: "tok"}
	if _, err := p.identityFromUserInfo(context.Background(), tok.toOAuth2()); err == nil {
		t.Fatal("sans subject devrait retourner une erreur")
	}
}

func TestIdentityFromUserInfo_EmailViaRepli(t *testing.T) {
	// Premier serveur : userinfo sans email.
	userInfo := map[string]any{"id": float64(99)}
	emailsList := []map[string]any{{"email": "hidden@github.dev", "primary": true, "verified": true}}

	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userInfo)
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(emailsList)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := &Provider{
		name:        "github",
		userInfoURL: srv.URL + "/user",
		userInfoMap: defs["github"].userInfoMap,
		emailsURL:   srv.URL + "/user/emails",
	}
	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	tok := &fakeOAuthToken{access: "tok"}
	id, err := p.identityFromUserInfo(context.Background(), tok.toOAuth2())
	if err != nil {
		t.Fatalf("identityFromUserInfo (repli email): %v", err)
	}
	if id.Email != "hidden@github.dev" {
		t.Errorf("Email = %q, attendu 'hidden@github.dev'", id.Email)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// fakeOAuthToken wraps an access token to produce an *oauth2.Token.
type fakeOAuthToken struct {
	access string
}

func (f *fakeOAuthToken) toOAuth2() *oauth2.Token {
	return &oauth2.Token{AccessToken: f.access}
}
