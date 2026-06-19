package service

import (
	"database/sql"
	"errors"
	"testing"
)

// ─── suffixedUsername ─────────────────────────────────────────────────────────

func TestSuffixedUsername_Format(t *testing.T) {
	result := suffixedUsername("alice")
	// doit contenir _ et avoir une longueur raisonnable
	found := false
	for _, c := range result {
		if c == '_' {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("suffixedUsername(%q) = %q, attendu un underscore", "alice", result)
	}
	if len(result) > 50 {
		t.Fatalf("suffixedUsername trop long : %d > 50", len(result))
	}
}

func TestSuffixedUsername_TroncatureBase(t *testing.T) {
	// Base très longue : le suffixe ne doit pas dépasser 50 chars au total
	longBase := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmn"
	result := suffixedUsername(longBase)
	if len(result) > 50 {
		t.Fatalf("suffixedUsername(longBase) len = %d > 50", len(result))
	}
}

func TestSuffixedUsername_Unicité(t *testing.T) {
	a := suffixedUsername("alice")
	b := suffixedUsername("alice")
	if a == b {
		t.Fatal("deux appels suffixedUsername doivent être différents (suffixe aléatoire)")
	}
}

// ─── randomHex ────────────────────────────────────────────────────────────────

func TestRandomHex_Longueur(t *testing.T) {
	for _, n := range []int{4, 8, 16} {
		got := randomHex(n)
		if len(got) != n {
			t.Fatalf("randomHex(%d) len = %d, attendu %d", n, len(got), n)
		}
	}
}

func TestRandomHex_Unicité(t *testing.T) {
	a := randomHex(8)
	b := randomHex(8)
	if a == b {
		t.Fatal("randomHex doit retourner des valeurs différentes à chaque appel")
	}
}

func TestRandomHex_FormatHex(t *testing.T) {
	h := randomHex(8)
	for _, c := range h {
		if !(('0' <= c && c <= '9') || ('a' <= c && c <= 'f')) {
			t.Fatalf("randomHex contient %c (non-hex)", c)
		}
	}
}

// ─── isUniqueViolation ────────────────────────────────────────────────────────

func TestIsUniqueViolation_Nil(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Fatal("nil n'est pas une violation UNIQUE")
	}
}

type fakePG struct{ code string }

func (e *fakePG) Error() string    { return "pg " + e.code }
func (e *fakePG) SQLState() string { return e.code }

func TestIsUniqueViolation_23505(t *testing.T) {
	if !isUniqueViolation(&fakePG{code: "23505"}) {
		t.Fatal("23505 doit être reconnu")
	}
}

func TestIsUniqueViolation_AutreCode(t *testing.T) {
	if isUniqueViolation(&fakePG{code: "23502"}) {
		t.Fatal("23502 ne doit pas être une violation UNIQUE")
	}
}

// ─── mapDetails ──────────────────────────────────────────────────────────────

func TestMapDetails_NoRows(t *testing.T) {
	_, err := mapDetails(nil, sql.ErrNoRows)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("mapDetails(ErrNoRows) = %v, attendu ErrUserNotFound", err)
	}
}

func TestMapDetails_AutreErreur(t *testing.T) {
	sentinel := errors.New("db down")
	_, err := mapDetails(nil, sentinel)
	if err == nil {
		t.Fatal("une erreur DB doit être propagée")
	}
}

func TestMapDetails_Succès(t *testing.T) {
	// mapDetails nil err → retourne le detail tel quel
	_, err := mapDetails(nil, nil)
	if err != nil {
		t.Fatalf("mapDetails(nil, nil) = %v, attendu nil", err)
	}
}
