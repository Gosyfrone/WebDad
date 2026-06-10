package oauth

import (
	"context"
	"testing"
)

// NewState doit produire des valeurs non vides et uniques (anti-CSRF).
func TestNewState(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		s, err := NewState()
		if err != nil {
			t.Fatalf("NewState a échoué : %v", err)
		}
		if s == "" {
			t.Fatal("NewState a renvoyé une chaîne vide")
		}
		if _, dup := seen[s]; dup {
			t.Fatalf("state dupliqué : %q", s)
		}
		seen[s] = struct{}{}
	}
}

// Get renvoie ErrUnknownProvider pour un provider absent OU non configuré
// (ClientID vide), sans tenter de discovery réseau.
func TestRegistryGet_UnknownOrUnconfigured(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{
		RedirectBaseURL: "http://localhost:3000/",
		Providers: map[string]Credentials{
			"google":    {ClientID: ""}, // non configuré → ignoré
			"microsoft": {ClientID: "", ClientSecret: "x"},
		},
	})

	for _, name := range []string{"google", "microsoft", "facebook"} {
		if _, err := reg.Get(name); err != ErrUnknownProvider {
			t.Errorf("Get(%q) : attendu ErrUnknownProvider, obtenu %v", name, err)
		}
	}
}

// NewRegistry normalise la base du redirect URI (slash final retiré).
func TestNewRegistry_TrimsRedirectBase(t *testing.T) {
	reg := NewRegistry(context.Background(), Options{RedirectBaseURL: "http://localhost:3000/"})
	if reg.redirectBase != "http://localhost:3000" {
		t.Errorf("redirectBase = %q, attendu sans slash final", reg.redirectBase)
	}
}
