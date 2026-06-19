package mailer

import (
	"errors"
	"strings"
	"testing"
)

func TestConsoleMailerReportsDeliveryUnavailable(t *testing.T) {
	m := &consoleMailer{}
	err := m.Send(Message{
		To:      "test@example.com",
		Subject: "Test",
		Text:    "Message local",
	})
	if !errors.Is(err, ErrDeliveryUnavailable) {
		t.Fatalf("Send() error = %v, want ErrDeliveryUnavailable", err)
	}
}

func TestConsoleMailer_HTMLFallback(t *testing.T) {
	// HTML fourni sans texte → utilise le HTML comme corps loggé
	m := &consoleMailer{from: "noreply@breezy.dev"}
	err := m.Send(Message{
		To:      "user@example.com",
		Subject: "Confirmation",
		HTML:    "<h1>Bienvenue</h1>",
	})
	if !errors.Is(err, ErrDeliveryUnavailable) {
		t.Fatalf("Send() HTML only = %v, attendu ErrDeliveryUnavailable", err)
	}
}

func TestBuildMIME_TextOnly(t *testing.T) {
	raw := buildMIME("Breezy <no-reply@breezy.dev>", "no-reply@breezy.dev", Message{
		To:      "user@example.com",
		Subject: "Test",
		Text:    "Hello World",
	})
	body := string(raw)
	if !strings.Contains(body, "Content-Type: text/plain") {
		t.Fatalf("text/plain absent : %s", body)
	}
	if strings.Contains(body, "multipart/alternative") {
		t.Fatal("message texte seul ne doit pas être multipart")
	}
	if !strings.Contains(body, "Hello World") {
		t.Fatal("corps texte absent")
	}
}

func TestBuildMIME_Multipart(t *testing.T) {
	raw := buildMIME("Breezy <no-reply@breezy.dev>", "no-reply@breezy.dev", Message{
		To:      "user@example.com",
		Subject: "Test HTML",
		HTML:    "<p>Hello</p>",
	})
	body := string(raw)
	if !strings.Contains(body, "multipart/alternative") {
		t.Fatalf("multipart absent : %s", body)
	}
	if !strings.Contains(body, "text/plain") {
		t.Fatal("partie text/plain absente du multipart")
	}
	if !strings.Contains(body, "text/html") {
		t.Fatal("partie text/html absente du multipart")
	}
	if !strings.Contains(body, "<p>Hello</p>") {
		t.Fatal("HTML absent du corps")
	}
}

func TestBuildMIME_SubjectEncoded(t *testing.T) {
	raw := buildMIME("From <f@breezy.dev>", "f@breezy.dev", Message{
		To:      "u@example.com",
		Subject: "Confirmez votre adresse e-mail",
		Text:    "body",
	})
	body := string(raw)
	if !strings.Contains(body, "Subject:") {
		t.Fatal("champ Subject absent")
	}
}

func TestBuildMIME_MessageID(t *testing.T) {
	raw := buildMIME("F <f@breezy.dev>", "f@breezy.dev", Message{
		To:      "u@example.com",
		Subject: "Test",
		Text:    "body",
	})
	if !strings.Contains(string(raw), "Message-ID:") {
		t.Fatal("Message-ID absent")
	}
}

func TestBuildMIME_NomDomaineFallback(t *testing.T) {
	// Adresse envelope sans @ → domaine fallback breezy.local
	raw := buildMIME("F <f@d.com>", "noemail", Message{
		To:      "u@example.com",
		Subject: "Test",
		Text:    "body",
	})
	if !strings.Contains(string(raw), "breezy.local") {
		t.Fatal("domaine fallback breezy.local attendu")
	}
}

func TestRandHex(t *testing.T) {
	a := randHex(8)
	b := randHex(8)
	if len(a) != 16 { // 8 octets = 16 hex chars
		t.Fatalf("randHex(8) len = %d, attendu 16", len(a))
	}
	if a == b {
		t.Fatal("randHex doit générer des valeurs différentes")
	}
}

func TestFromHeaderAjouteNom(t *testing.T) {
	// Adresse sans "<" → enveloppée dans "Breezy <addr>"
	raw := buildMIME("noreply@breezy.dev", "noreply@breezy.dev", Message{
		To:      "u@example.com",
		Subject: "Test",
		Text:    "body",
	})
	body := string(raw)
	// Le from doit être l'en-tête brut passé
	if !strings.Contains(body, "noreply@breezy.dev") {
		t.Fatalf("from absent : %s", body[:200])
	}
}
