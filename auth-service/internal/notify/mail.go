// Package notify porte le client serveur-à-serveur vers le mail-service.
// auth-service y délègue l'envoi des e-mails transactionnels (vérification
// d'adresse) en best-effort : une panne du mail-service ne casse jamais
// l'inscription ni la connexion.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MailClient appelle POST {baseURL}/internal/send avec le secret partagé
// (en-tête X-Internal-Secret). Hors gateway, symétrique de l'émission
// d'événements vers notification-service.
type MailClient struct {
	baseURL string
	secret  string
	http    *http.Client
}

// NewMailClient construit le client avec un timeout court (best-effort).
func NewMailClient(baseURL, secret string) *MailClient {
	return &MailClient{
		baseURL: baseURL,
		secret:  secret,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// sendPayload : corps JSON attendu par mail-service POST /internal/send.
type sendPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html,omitempty"`
	Text    string `json:"text,omitempty"`
}

// Send transmet un e-mail au mail-service. Renvoie une erreur que l'appelant
// logge sans la propager (best-effort).
func (c *MailClient) Send(to, subject, html, text string) error {
	body, err := json.Marshal(sendPayload{To: to, Subject: subject, HTML: html, Text: text})
	if err != nil {
		return fmt.Errorf("sérialisation mail : %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("construction requête mail : %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", c.secret)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("appel mail-service : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mail-service a répondu %d", resp.StatusCode)
	}
	return nil
}
