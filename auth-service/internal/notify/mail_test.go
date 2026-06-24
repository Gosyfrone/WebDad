package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── NewMailClient ────────────────────────────────────────────────────────────

func TestNewMailClient_ChampsCorrects(t *testing.T) {
	c := NewMailClient("http://mail:8089", "secret-test")
	if c.baseURL != "http://mail:8089" {
		t.Fatalf("baseURL = %q, attendu 'http://mail:8089'", c.baseURL)
	}
	if c.secret != "secret-test" {
		t.Fatalf("secret = %q, attendu 'secret-test'", c.secret)
	}
	if c.http == nil {
		t.Fatal("http client ne doit pas être nil")
	}
}

// ─── Send ────────────────────────────────────────────────────────────────────

func TestSend_TransmetPayloadCorrect(t *testing.T) {
	var received struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		HTML    string `json:"html"`
		Text    string `json:"text"`
	}
	var gotSecret string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Internal-Secret")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewMailClient(srv.URL, "my-secret")
	err := c.Send("alice@breezy.dev", "Bienvenue", "<b>Hi</b>", "Hi")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if gotSecret != "my-secret" {
		t.Fatalf("X-Internal-Secret = %q, attendu 'my-secret'", gotSecret)
	}
	if received.To != "alice@breezy.dev" {
		t.Fatalf("To = %q, attendu 'alice@breezy.dev'", received.To)
	}
	if received.Subject != "Bienvenue" {
		t.Fatalf("Subject = %q, attendu 'Bienvenue'", received.Subject)
	}
	if received.HTML != "<b>Hi</b>" {
		t.Fatalf("HTML = %q, attendu '<b>Hi</b>'", received.HTML)
	}
}

func TestSend_ServeurErreur_RetourneErreur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewMailClient(srv.URL, "s")
	err := c.Send("a@b.com", "sub", "", "text")
	if err == nil {
		t.Fatal("serveur 500 devrait retourner une erreur")
	}
}

func TestSend_URLInjoignable_RetourneErreur(t *testing.T) {
	c := NewMailClient("http://127.0.0.1:1", "s")
	err := c.Send("a@b.com", "sub", "", "text")
	if err == nil {
		t.Fatal("URL injoignable devrait retourner une erreur")
	}
}

func TestSend_URLInvalide_RetourneErreur(t *testing.T) {
	c := NewMailClient("://bad-url", "s")
	if err := c.Send("a@b.com", "sub", "", "text"); err == nil {
		t.Fatal("URL invalide devrait retourner une erreur")
	}
}
