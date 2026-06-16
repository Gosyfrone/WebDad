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
