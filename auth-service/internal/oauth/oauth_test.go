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

// checkMicrosoftIssuer n'accepte qu'un issuer Microsoft de forme stricte dont
// le tenant correspond au claim `tid` (comparaison insensible à la casse).
func TestCheckMicrosoftIssuer(t *testing.T) {
	const tid = "9188040d-6c67-4c5b-b112-36a304b66dad" // tenant des comptes perso (MSA)
	valid := "https://login.microsoftonline.com/" + tid + "/v2.0"

	cases := []struct {
		name    string
		iss     string
		tid     string
		wantErr bool
	}{
		{"tenant valide", valid, tid, false},
		{"casse différente", valid, "9188040D-6C67-4C5B-B112-36A304B66DAD", false},
		{"tid incohérent", valid, "00000000-0000-0000-0000-000000000000", true},
		{"tid absent", valid, "", true},
		{"mauvais host", "https://evil.example.com/" + tid + "/v2.0", tid, true},
		{"placeholder non résolu", microsoftIssuerPlaceholder, tid, true},
		{"v1.0 refusé", "https://login.microsoftonline.com/" + tid + "/v1.0", tid, true},
		{"suffixe parasite", valid + "/extra", tid, true},
	}
	for _, c := range cases {
		err := checkMicrosoftIssuer(c.iss, c.tid)
		if (err != nil) != c.wantErr {
			t.Errorf("%s : checkMicrosoftIssuer(%q, %q) = %v, wantErr=%v",
				c.name, c.iss, c.tid, err, c.wantErr)
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
