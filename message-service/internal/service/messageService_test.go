package service

import (
	"testing"

	"github.com/webdad/message-service/internal/models"
)

func TestSortedPair_OrderIndependent(t *testing.T) {
	a := sortedPair("bob", "alice")
	b := sortedPair("alice", "bob")
	if a[0] != "alice" || a[1] != "bob" {
		t.Fatalf("attendu [alice bob], obtenu %v", a)
	}
	if a[0] != b[0] || a[1] != b[1] {
		t.Fatalf("sortedPair doit être indépendant de l'ordre : %v vs %v", a, b)
	}
}

func TestDMKey_SameForBothDirections(t *testing.T) {
	if dmKey("u1", "u2") != dmKey("u2", "u1") {
		t.Fatalf("dmKey doit être symétrique")
	}
	if dmKey("u1", "u2") != "u1:u2" {
		t.Fatalf("dmKey attendu u1:u2, obtenu %q", dmKey("u1", "u2"))
	}
}

func TestCanWrite(t *testing.T) {
	cases := map[string]bool{
		models.MemberOwner:  true,
		models.MemberAdmin:  true,
		models.MemberTalker: true,
		models.MemberViewer: false, // lecture seule (communautés)
		"":                  false,
		"random":            false,
	}
	for role, want := range cases {
		if got := canWrite(role); got != want {
			t.Errorf("canWrite(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestClampLimit(t *testing.T) {
	if clampLimit(0) != DefaultLimit {
		t.Errorf("0 doit retomber sur le défaut (%d)", DefaultLimit)
	}
	if clampLimit(-5) != DefaultLimit {
		t.Errorf("négatif doit retomber sur le défaut")
	}
	if clampLimit(1000) != MaxLimit {
		t.Errorf("au-delà de Max doit être borné à %d", MaxLimit)
	}
	if clampLimit(10) != 10 {
		t.Errorf("une valeur valide doit être conservée")
	}
}

func TestParseID_RejectsInvalid(t *testing.T) {
	if _, err := parseID("not-an-objectid"); err != ErrInvalidID {
		t.Errorf("un id invalide doit donner ErrInvalidID, obtenu %v", err)
	}
}

func TestCheckRemoval(t *testing.T) {
	cases := []struct {
		name      string
		role      string
		isSelf    bool
		wantError error
	}{
		{"un membre peut quitter", models.MemberTalker, true, nil},
		{"l'owner ne peut PAS quitter", models.MemberOwner, true, ErrOwnerCannotLeave},
		{"l'owner peut exclure autrui", models.MemberOwner, false, nil},
		{"un membre ne peut PAS exclure autrui", models.MemberTalker, false, ErrOwnerOnly},
	}
	for _, tc := range cases {
		if got := checkRemoval(tc.role, tc.isSelf); got != tc.wantError {
			t.Errorf("%s : checkRemoval(%q,%v) = %v, want %v", tc.name, tc.role, tc.isSelf, got, tc.wantError)
		}
	}
}

func TestMaxGroupMembers(t *testing.T) {
	// La règle métier (cap groupe) doit valoir 32 (exigence produit).
	if MaxGroupMembers != 32 {
		t.Errorf("MaxGroupMembers attendu 32, obtenu %d", MaxGroupMembers)
	}
}
