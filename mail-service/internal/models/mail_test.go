package models

import "testing"

func TestSendRequest_ChampsVides(t *testing.T) {
	// SendRequest est une struct de binding. Aucune logique pure, on vérifie
	// juste que la struct peut être instanciée correctement.
	req := SendRequest{
		To:      "alice@breezy.dev",
		Subject: "Bienvenue",
		HTML:    "<b>Bonjour</b>",
		Text:    "Bonjour",
	}
	if req.To != "alice@breezy.dev" {
		t.Fatalf("To = %q", req.To)
	}
	if req.Subject != "Bienvenue" {
		t.Fatalf("Subject = %q", req.Subject)
	}
}
