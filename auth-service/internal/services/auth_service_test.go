package services

import (
	"testing"
)

// ─── validRole ───────────────────────────────────────────────────────────────

func TestValidRole(t *testing.T) {
	cases := []struct {
		role string
		want bool
	}{
		{"user", true},
		{"moderator", true},
		{"admin", true},
		{"", false},
		{"superadmin", false},
		{"User", false}, // sensible à la casse (JWT émet en minuscules)
		{"ADMIN", false},
	}
	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			if got := validRole(tc.role); got != tc.want {
				t.Fatalf("validRole(%q) = %v, attendu %v", tc.role, got, tc.want)
			}
		})
	}
}

// ─── hashToken ───────────────────────────────────────────────────────────────

func TestHashToken_Deterministe(t *testing.T) {
	a := hashToken("my-refresh-token")
	b := hashToken("my-refresh-token")
	if a != b {
		t.Fatal("hashToken doit être déterministe")
	}
}

func TestHashToken_Différents(t *testing.T) {
	a := hashToken("token-A")
	b := hashToken("token-B")
	if a == b {
		t.Fatal("des tokens différents doivent produire des hash différents")
	}
}

func TestHashToken_FormatHex(t *testing.T) {
	h := hashToken("test")
	if len(h) != 64 { // SHA-256 → 32 octets → 64 hex chars
		t.Fatalf("hashToken len = %d, attendu 64 (SHA-256 hex)", len(h))
	}
	for _, c := range h {
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') {
			t.Fatalf("hashToken contient un caractère non-hex : %c", c)
		}
	}
}

func TestHashToken_StringVide(t *testing.T) {
	h := hashToken("")
	if len(h) != 64 {
		t.Fatalf("hashToken('') len = %d, attendu 64", len(h))
	}
}

// ─── randomToken ─────────────────────────────────────────────────────────────

func TestRandomToken_UniqueEtNonVide(t *testing.T) {
	a, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken() erreur : %v", err)
	}
	if a == "" {
		t.Fatal("randomToken() retourne une chaîne vide")
	}
	b, _ := randomToken()
	if a == b {
		t.Fatal("deux appels randomToken() doivent retourner des valeurs différentes")
	}
}

func TestRandomToken_Longueur(t *testing.T) {
	tok, _ := randomToken()
	// base64.RawURLEncoding de 32 octets = 43 caractères
	if len(tok) < 40 {
		t.Fatalf("randomToken trop court : %d chars", len(tok))
	}
}

// ─── isUniqueViolation ────────────────────────────────────────────────────────

func TestIsUniqueViolation_ErreurNil(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Fatal("nil ne doit pas être une violation UNIQUE")
	}
}

type fakePGErr struct{ code string }

func (e *fakePGErr) Error() string    { return "pg error " + e.code }
func (e *fakePGErr) SQLState() string { return e.code }

func TestIsUniqueViolation_23505(t *testing.T) {
	err := &fakePGErr{code: "23505"}
	if !isUniqueViolation(err) {
		t.Fatal("code 23505 doit être reconnu comme violation UNIQUE")
	}
}

func TestIsUniqueViolation_AutreCode(t *testing.T) {
	err := &fakePGErr{code: "23503"} // foreign key, pas UNIQUE
	if isUniqueViolation(err) {
		t.Fatal("code 23503 ne doit pas être une violation UNIQUE")
	}
}

func TestIsUniqueViolation_ErreurSansCode(t *testing.T) {
	err := &plainErr{msg: "some error"}
	if isUniqueViolation(err) {
		t.Fatal("erreur sans SQLState ne doit pas être une violation UNIQUE")
	}
}

type plainErr struct{ msg string }

func (e *plainErr) Error() string { return e.msg }
