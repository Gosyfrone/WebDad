package service

import (
	"testing"
	"time"

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

func TestMaxTalkers(t *testing.T) {
	// La règle métier (cap des participants pouvant écrire) doit valoir 32.
	if MaxTalkers != 32 {
		t.Errorf("MaxTalkers attendu 32, obtenu %d", MaxTalkers)
	}
}

func TestIsManageable(t *testing.T) {
	cases := map[string]bool{
		models.TypeGroup:     true,
		models.TypeCommunity: true,
		models.TypeDM:        false, // un DM n'a ni admin de membres ni suppression
		"":                   false,
	}
	for typ, want := range cases {
		if got := isManageable(typ); got != want {
			t.Errorf("isManageable(%q) = %v, want %v", typ, got, want)
		}
	}
}

func TestCanWrite_ViewerCannot(t *testing.T) {
	// Garde-fou communautés : un viewer ne peut jamais écrire.
	if canWrite(models.MemberViewer) {
		t.Error("un viewer ne doit PAS pouvoir écrire")
	}
	if !canWrite(models.MemberTalker) {
		t.Error("un talker doit pouvoir écrire")
	}
}

func TestConvLess_PinnedFirstThenActivity(t *testing.T) {
	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	pinOld := t0.Add(1 * time.Hour)
	pinNew := t0.Add(2 * time.Hour)

	view := func(updated time.Time, pinnedAt *time.Time) models.ConversationView {
		return models.ConversationView{UpdatedAt: updated, PinnedAt: pinnedAt}
	}

	pinnedRecent := view(t0, &pinNew)
	pinnedOlder := view(t0.Add(5*time.Hour), &pinOld) // plus actif mais épinglé plus tôt
	active := view(t0.Add(9*time.Hour), nil)          // non épinglé, très actif
	stale := view(t0, nil)                            // non épinglé, peu actif

	// Une épinglée passe toujours devant une non-épinglée, même moins active.
	if !convLess(pinnedOlder, active) {
		t.Error("une conversation épinglée doit passer devant une non-épinglée")
	}
	if convLess(active, pinnedOlder) {
		t.Error("une non-épinglée ne doit pas passer devant une épinglée")
	}
	// Entre deux épinglées : la plus récemment épinglée d'abord.
	if !convLess(pinnedRecent, pinnedOlder) {
		t.Error("l'épinglage le plus récent doit passer en premier")
	}
	// Entre deux non-épinglées : la plus active d'abord.
	if !convLess(active, stale) {
		t.Error("la non-épinglée la plus active doit passer en premier")
	}
}

func TestMentionedTargets(t *testing.T) {
	members := []string{"u1", "u2", "u3"}

	cases := []struct {
		name      string
		mentioned []string
		sender    string
		want      []string
	}{
		{"garde les membres réels hors expéditeur", []string{"u2", "u3"}, "u1", []string{"u2", "u3"}},
		{"ignore l'expéditeur", []string{"u1", "u2"}, "u1", []string{"u2"}},
		{"ignore les non-membres", []string{"u2", "ghost"}, "u1", []string{"u2"}},
		{"déduplique", []string{"u2", "u2"}, "u1", []string{"u2"}},
		{"ignore les vides", []string{"", "u3"}, "u1", []string{"u3"}},
		{"aucune mention → nil", nil, "u1", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mentionedTargets(tc.mentioned, members, tc.sender)
			if len(got) != len(tc.want) {
				t.Fatalf("mentionedTargets = %v ; attendu %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("mentionedTargets = %v ; attendu %v", got, tc.want)
				}
			}
		})
	}
}
