// Package mailer envoie des e-mails. Deux transports interchangeables, choisis
// au démarrage selon la présence d'une config SMTP :
//   - smtpMailer    : production — Gmail via net/smtp (STARTTLS auto sur 587),
//     stdlib uniquement (aucune dépendance supplémentaire) ;
//   - consoleMailer : dev (SMTP non configuré) — l'e-mail est loggé sur stdout
//     pour pouvoir cliquer le lien depuis `make logs-mail` sans Gmail.
//
// Le transport est content-agnostic (l'appelant fournit le corps déjà rendu),
// à l'image du media-service qui stocke des octets opaques.
package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/webdad/mail-service/internal/config"
)

// Message : un e-mail à envoyer. Au moins l'un de HTML/Text est attendu ;
// HTML prime s'il est présent.
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Mailer abstrait le transport pour permettre le repli console en dev.
type Mailer interface {
	Send(msg Message) error
}

// New choisit le transport selon la présence de la config SMTP et logge le
// mode retenu au démarrage.
func New(cfg *config.Config) Mailer {
	if cfg.SMTP.Configured() {
		log.Printf("[mailer] transport SMTP actif (%s:%s)", cfg.SMTP.Host, cfg.SMTP.Port)
		return &smtpMailer{smtp: cfg.SMTP}
	}
	log.Print("[mailer] SMTP non configuré → transport CONSOLE (les e-mails sont loggés, pas envoyés)")
	return &consoleMailer{from: cfg.SMTP.From}
}

// ─── Transport console (dev) ─────────────────────────────────────

type consoleMailer struct{ from string }

func (m *consoleMailer) Send(msg Message) error {
	body := msg.Text
	if body == "" {
		body = msg.HTML
	}
	from := m.from
	if from == "" {
		from = "no-reply@breezy.local"
	}
	log.Printf("[mailer:console] e-mail (NON envoyé)\n  From   : %s\n  To     : %s\n  Subject: %s\n  ---\n%s\n  ---",
		from, msg.To, msg.Subject, body)
	return nil
}

// ─── Transport SMTP (prod) ───────────────────────────────────────

type smtpMailer struct{ smtp config.SMTP }

func (m *smtpMailer) Send(msg Message) error {
	addr := m.smtp.Host + ":" + m.smtp.Port
	auth := smtp.PlainAuth("", m.smtp.User, m.smtp.Password, m.smtp.Host)

	// Enveloppe : Gmail réécrit le From sur le compte authentifié, on utilise
	// donc l'utilisateur SMTP comme expéditeur d'enveloppe. L'en-tête From
	// affiché reprend SMTP_FROM si fourni, sinon l'utilisateur.
	fromHeader := m.smtp.From
	if fromHeader == "" {
		fromHeader = m.smtp.User
	}

	contentType := `text/plain; charset="utf-8"`
	body := msg.Text
	if msg.HTML != "" {
		contentType = `text/html; charset="utf-8"`
		body = msg.HTML
	}

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", msg.Subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: %s\r\n", contentType)
	b.WriteString("\r\n")
	b.WriteString(body)

	// smtp.SendMail négocie STARTTLS automatiquement si le serveur l'annonce
	// (cas de Gmail sur le port 587).
	if err := smtp.SendMail(addr, auth, m.smtp.User, []string{msg.To}, []byte(b.String())); err != nil {
		return fmt.Errorf("envoi SMTP : %w", err)
	}
	return nil
}
