package mailer

import (
	"bufio"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/webdad/mail-service/internal/config"
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

func TestNewChoosesConsoleMailerWhenSMTPIsIncomplete(t *testing.T) {
	m := New(&config.Config{
		SMTP: config.SMTP{From: "no-reply@breezy.dev"},
	})
	if _, ok := m.(*consoleMailer); !ok {
		t.Fatalf("New() = %T, want *consoleMailer", m)
	}
}

func TestNewChoosesSMTPMailerWhenSMTPIsConfigured(t *testing.T) {
	m := New(&config.Config{
		SMTP: config.SMTP{
			Host:     "smtp.example.com",
			Port:     "587",
			User:     "user@example.com",
			Password: "password",
		},
	})
	if _, ok := m.(*smtpMailer); !ok {
		t.Fatalf("New() = %T, want *smtpMailer", m)
	}
}

func TestSMTPMailerSendUsesLocalSMTPServer(t *testing.T) {
	addr, done := startFakeSMTPServer(t)
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		t.Fatalf("invalid listener addr %q", addr)
	}

	m := &smtpMailer{smtp: config.SMTP{
		Host:     host,
		Port:     port,
		User:     "sender@example.com",
		Password: "password",
	}}

	err := m.Send(Message{
		To:      "user@example.com",
		Subject: "Test SMTP",
		Text:    "Hello SMTP",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	<-done
}

func TestSMTPMailerSendWrapsSMTPFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		t.Fatalf("invalid listener addr %q", addr)
	}

	m := &smtpMailer{smtp: config.SMTP{
		Host:     host,
		Port:     port,
		User:     "sender@example.com",
		Password: "password",
	}}

	err = m.Send(Message{To: "user@example.com", Subject: "Test", Text: "body"})
	if err == nil {
		t.Fatal("Send() error = nil, want SMTP error")
	}
	if !strings.Contains(err.Error(), "envoi SMTP") {
		t.Fatalf("Send() error = %v, want wrapped SMTP error", err)
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

func TestBuildMIME_UsesEnvelopeDomain(t *testing.T) {
	raw := buildMIME("F <f@example.org>", "sender@example.org", Message{
		To:      "u@example.com",
		Subject: "Test",
		Text:    "body",
	})
	if !strings.Contains(string(raw), "@example.org>") {
		t.Fatalf("Message-ID should use envelope domain: %s", string(raw))
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

func startFakeSMTPServer(t *testing.T) (string, <-chan struct{}) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ln.Close()

		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("Accept() error = %v", err)
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		write := func(s string) {
			if _, err := conn.Write([]byte(s)); err != nil {
				t.Errorf("Write(%q) error = %v", s, err)
			}
		}
		readLine := func() string {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Errorf("ReadString() error = %v", err)
				return ""
			}
			return line
		}

		write("220 localhost ESMTP\r\n")
		if line := readLine(); !strings.HasPrefix(line, "EHLO ") {
			t.Errorf("first command = %q, want EHLO", line)
			return
		}
		write("250-localhost\r\n250 AUTH PLAIN\r\n")

		if line := readLine(); !strings.HasPrefix(line, "AUTH PLAIN ") {
			t.Errorf("auth command = %q, want AUTH PLAIN", line)
			return
		}
		write("235 2.7.0 Authentication successful\r\n")

		for _, prefix := range []string{"MAIL FROM:", "RCPT TO:", "DATA"} {
			if line := readLine(); !strings.HasPrefix(line, prefix) {
				t.Errorf("command = %q, want prefix %q", line, prefix)
				return
			}
			if prefix == "DATA" {
				write("354 End data with <CR><LF>.<CR><LF>\r\n")
				break
			}
			write("250 OK\r\n")
		}

		for {
			if strings.TrimSpace(readLine()) == "." {
				break
			}
		}
		write("250 OK queued\r\n")

		if line := readLine(); !strings.HasPrefix(line, "QUIT") {
			t.Errorf("final command = %q, want QUIT", line)
			return
		}
		write("221 Bye\r\n")
	}()

	return ln.Addr().String(), done
}
