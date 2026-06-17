// Package notifier émet des événements vers le notification-service après une
// action de messagerie (nouveau message, mention d'un membre dans un message).
// Communication serveur-à-serveur sur le réseau Docker : best-effort et
// fire-and-forget (un échec n'impacte JAMAIS l'envoi du message).
//
// Le message-service reste AVEUGLE au contenu (chiffré côté client) : le client
// lui fournit les ids des membres mentionnés (métadonnée d'appartenance, jamais
// le texte). On émet donc un événement `message_mention` par destinataire, avec
// l'id de conversation (navigation) — la résolution @handle→id a déjà eu lieu
// côté client. Le contrat JSON est aligné sur `models.Event` du
// notification-service.
package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Types d'événements (alignés sur le notification-service).
const (
	TypeMessage        = "message"
	TypeMessageMention = "message_mention"
)

// Event — charge utile envoyée au notification-service.
type Event struct {
	Type           string `json:"type"`
	ActorID        string `json:"actor_id"`
	RecipientID    string `json:"recipient_id"`
	ConversationID string `json:"conversation_id"`
}

// Notifier émet des événements (interface → service testable avec un faux/no-op).
type Notifier interface {
	Emit(ev Event)
}

// Noop ne fait rien : utilisé quand le notification-service n'est pas configuré
// (message-service reste autonome) ou dans les tests.
type Noop struct{}

func (Noop) Emit(Event) {}

// HTTPNotifier poste les événements au notification-service.
type HTTPNotifier struct {
	url    string // ex. http://notification-service:8086/internal/events
	secret string
	client *http.Client
}

// New construit un notifier HTTP. `baseURL` est la base du notification-service
// (l'endpoint /internal/events est ajouté).
func New(baseURL, secret string) *HTTPNotifier {
	return &HTTPNotifier{
		url:    baseURL + "/internal/events",
		secret: secret,
		client: &http.Client{Timeout: 4 * time.Second},
	}
}

// Emit envoie l'événement en arrière-plan (fire-and-forget). On ne bloque ni ne
// fait échouer l'appelant : une indisponibilité du notification-service ne doit
// pas casser l'envoi d'un message.
func (n *HTTPNotifier) Emit(ev Event) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		body, err := json.Marshal(ev)
		if err != nil {
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Secret", n.secret)

		resp, err := n.client.Do(req)
		if err != nil {
			log.Printf("[notifier] envoi de l'événement %q : %v", ev.Type, err)
			return
		}
		_ = resp.Body.Close()
	}()
}
