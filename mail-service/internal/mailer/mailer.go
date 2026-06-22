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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/webdad/mail-service/internal/config"
)

// ErrDeliveryUnavailable indique qu'aucun transport capable de remettre le
// message n'est configuré. Le transport console conserve le rendu local du
// mail, mais ne doit jamais être assimilé à un envoi réussi.
var ErrDeliveryUnavailable = errors.New("transport e-mail non configuré")

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
	return ErrDeliveryUnavailable
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
	// Nom d'affichage si l'adresse est nue : un From nominatif est mieux noté.
	if !strings.Contains(fromHeader, "<") {
		fromHeader = "Breezy <" + fromHeader + ">"
	}

	raw := buildMIME(fromHeader, m.smtp.User, msg)

	// smtp.SendMail négocie STARTTLS automatiquement si le serveur l'annonce
	// (cas de Gmail sur le port 587).
	if err := smtp.SendMail(addr, auth, m.smtp.User, []string{msg.To}, raw); err != nil {
		return fmt.Errorf("envoi SMTP : %w", err)
	}
	return nil
}

var htmlTagRE = regexp.MustCompile(`<[^>]*>`)
var readRandom = rand.Read

// buildMIME assemble un message RFC 5322 complet :
//   - en-têtes Date + Message-ID (leur absence est un signal anti-spam) ;
//   - Subject encodé RFC 2047 (accents/emoji) ;
//   - corps multipart/alternative texte + HTML (un HTML seul sans repli texte
//     est lourdement pénalisé par les filtres anti-spam).
//
// Si seul le texte est fourni, le message reste un simple text/plain.
func buildMIME(fromHeader, envelopeFrom string, msg Message) []byte {
	domain := "breezy.local"
	if i := strings.LastIndex(envelopeFrom, "@"); i >= 0 && i+1 < len(envelopeFrom) {
		domain = envelopeFrom[i+1:]
	}

	text := msg.Text
	if text == "" && msg.HTML != "" {
		// Repli minimal : on dérive un texte lisible du HTML (tags retirés).
		text = strings.TrimSpace(htmlTagRE.ReplaceAllString(msg.HTML, ""))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", fromHeader)
	// Reply-To explicite (= From) : une adresse de réponse réelle et alignée
	// améliore la réputation et évite que les réponses partent dans le vide.
	fmt.Fprintf(&b, "Reply-To: %s\r\n", fromHeader)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", msg.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%d.%s@%s>\r\n", time.Now().UnixNano(), randHex(8), domain)
	b.WriteString("MIME-Version: 1.0\r\n")

	// HTML absent → simple texte.
	if msg.HTML == "" {
		b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
		b.WriteString(text)
		return []byte(b.String())
	}

	boundary := "breezy_" + randHex(16)
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)

	// Partie texte (priorité de repli pour les clients sans HTML / anti-spam).
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	b.WriteString(text)
	b.WriteString("\r\n\r\n")

	// Partie HTML.
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
	b.WriteString(msg.HTML)
	b.WriteString("\r\n\r\n")

	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return []byte(b.String())
}

// randHex retourne n octets aléatoires en hexadécimal (boundary, Message-ID).
func randHex(n int) string {
	buf := make([]byte, n)
	if _, err := readRandom(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
