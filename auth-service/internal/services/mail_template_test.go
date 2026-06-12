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
		"", // pas d'encart code sur ce mail
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

	// Sans code, l'encart monospace ne doit pas apparaître.
	if strings.Contains(out, "monospace") {
		t.Errorf("encart code rendu alors que code est vide")
	}
}

func TestBrandedEmailHTMLWithCode(t *testing.T) {
	out := brandedEmailHTML(
		"https://breezy.app/",
		"Ton compte est prêt",
		"Voici ton mot de passe.",
		"S3cr3t<Pass>",
		"Se connecter",
		"https://breezy.app/login",
		"Garde-le secret.",
	)

	if strings.Contains(out, "{{") {
		t.Fatalf("jeton de gabarit non substitué dans la sortie")
	}
	// L'encart monospace doit être rendu et la valeur échappée (pas de < brut).
	if !strings.Contains(out, "monospace") {
		t.Errorf("encart code absent alors que code est fourni")
	}
	if !strings.Contains(out, "S3cr3t&lt;Pass&gt;") {
		t.Errorf("valeur du code non échappée/absente dans l'encart")
	}
}
