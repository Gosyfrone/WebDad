package service

import (
	"errors"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	cases := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valide simple", "alice", false},
		{"valide avec chiffres et underscore", "alice_42", false},
		{"valide avec point interne", "jean.dupont", false},
		{"valide points multiples internes", "a.b.c", false},
		{"valide max", "a234567890123456789012345678901234567890123456789", false}, // 49
		{"trop court", "ab", true},
		{"trop long", "a2345678901234567890123456789012345678901234567890x", true}, // 51
		{"caractère interdit (tiret)", "alice-b", true},
		{"caractère interdit (espace)", "alice b", true},
		{"caractère interdit (accent)", "alicé", true},
		{"point en début", ".alice", true},
		{"point en fin", "alice.", true},
		{"point doublé", "jean..dupont", true},
		{"point seul", "...", true},
		{"vide", "", true},
		{"réservé me", "me", true},
		{"réservé admin (insensible à la casse)", "Admin", true},
		{"réservé users", "users", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUsername(tc.username)
			if tc.wantErr && err == nil {
				t.Fatalf("attendu une erreur pour %q, obtenu nil", tc.username)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("attendu nil pour %q, obtenu %v", tc.username, err)
			}
			if tc.wantErr && err != nil && !errors.Is(err, ErrInvalidUsername) {
				t.Fatalf("attendu ErrInvalidUsername, obtenu %v", err)
			}
		})
	}
}

func TestDefaultUsername(t *testing.T) {
	cases := []struct {
		email string
		want  string
	}{
		{"alice@breezy.dev", "alice"},
		{"Alice.B@breezy.dev", "aliceb"},    // point retiré, minusculisé
		{"a@x.com", "user"},                 // < 3 → fallback
		{"jean-pierre@x.com", "jeanpierre"}, // tiret retiré
		{"bob+tag@x.com", "bobtag"},         // + retiré
		{"weird", "weird"},                  // pas de @
	}
	for _, tc := range cases {
		t.Run(tc.email, func(t *testing.T) {
			if got := defaultUsername(tc.email); got != tc.want {
				t.Fatalf("defaultUsername(%q) = %q, attendu %q", tc.email, got, tc.want)
			}
		})
	}
}
