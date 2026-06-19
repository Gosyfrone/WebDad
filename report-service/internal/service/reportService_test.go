package service

import (
	"strings"
	"testing"

	"github.com/webdad/report-service/internal/models"
)

// TestReasonAndEntityTables vérifie que les tables de validation couvrent bien
// les valeurs attendues (motifs, types d'entité, statuts).
func TestReasonAndEntityTables(t *testing.T) {
	for _, r := range []string{
		models.ReasonInappropriate, models.ReasonOffensive,
		models.ReasonBug, models.ReasonSpam, models.ReasonOther,
	} {
		if !validReasons[r] {
			t.Errorf("motif %q devrait être valide", r)
		}
	}
	if validReasons["__bidon__"] {
		t.Error("un motif inconnu ne doit pas être valide (clé Mongo bornée)")
	}

	for _, e := range []string{
		models.EntityPost, models.EntityMessage,
		models.EntityGroupMessage, models.EntityProfile,
	} {
		if !validEntityTypes[e] {
			t.Errorf("type d'entité %q devrait être valide", e)
		}
	}
	if validEntityTypes[models.EntityApp] {
		t.Error("`app` est réservé aux bugs, pas une entité signalable directement")
	}

	for _, s := range []string{models.StatusOpen, models.StatusClosed, models.StatusReopened} {
		if !validStatuses[s] {
			t.Errorf("statut %q devrait être valide", s)
		}
	}
}

// TestTextLimits documente les bornes de longueur exigées (255 modération / 500 bug).
func TestTextLimits(t *testing.T) {
	if models.MaxModerationText != 255 {
		t.Errorf("limite modération = %d, attendu 255", models.MaxModerationText)
	}
	if models.MaxBugText != 500 {
		t.Errorf("limite bug = %d, attendu 500", models.MaxBugText)
	}

	// Le comptage se fait en runes (caractères), pas en octets : un texte de 255
	// caractères accentués reste valide même s'il pèse plus de 255 octets.
	accented := strings.Repeat("é", 255)
	if len([]rune(accented)) != 255 {
		t.Errorf("comptage runes = %d, attendu 255", len([]rune(accented)))
	}
}

// TestHasBlockingSanction couvre le garde-fou de la validation : une entité
// déjà sanctionnée (contenu retiré ou auteur averti) ne peut plus être jugée
// conforme. Les autres actions (réponse, changement de statut…) ne bloquent pas.
func TestHasBlockingSanction(t *testing.T) {
	cases := []struct {
		name  string
		types []string
		want  bool
	}{
		{"aucune action", nil, false},
		{"contenu retiré", []string{models.ActionContentRemoved}, true},
		{"auteur averti", []string{models.ActionWarned}, true},
		{"retrait après réponse", []string{models.ActionReply, models.ActionContentRemoved}, true},
		{"réponse seule", []string{models.ActionReply}, false},
		{"changement de statut seul", []string{models.ActionStatusChange}, false},
		{"auto-masquage seul ne bloque pas", []string{models.ActionAutoHidden}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actions := make([]models.Action, len(tc.types))
			for i, ty := range tc.types {
				actions[i] = models.Action{Type: ty}
			}
			if got := hasBlockingSanction(actions); got != tc.want {
				t.Errorf("hasBlockingSanction(%v) = %v, attendu %v", tc.types, got, tc.want)
			}
		})
	}
}
