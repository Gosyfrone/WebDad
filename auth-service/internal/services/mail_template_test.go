package services

import (
	"strings"
	"testing"
)

func TestBrandedEmailHTML(t *testing.T) {
	link := "https://breezy.app/verify-email?token=abc123"
	out := brandedEmailHTML(
		"https://breezy.app/",
		"Bienvenue sur Breezy",
		"Confirme ton adresse.",
		"Vérifier mon adresse",
		link,
		"Ce lien expire dans 24 heures.",
	)

	// Aucun jeton de gabarit ne doit subsister après substitution.
	if strings.Contains(out, "{{") {
		t.Fatalf("jeton de gabarit non substitué dans la sortie")
	}

	for _, want := range []string{
		"Bienvenue sur Breezy",               // heading
		"Confirme ton adresse.",              // intro / pré-en-tête
		"Vérifier mon adresse",               // libellé du bouton
		link,                                 // CTA + repli copiable
		"https://breezy.app/logo_breezy.png", // logo dérivé du baseURL (slash final normalisé)
		"Ce lien expire dans 24 heures.",     // footnote
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sortie e-mail sans %q", want)
		}
	}
}
