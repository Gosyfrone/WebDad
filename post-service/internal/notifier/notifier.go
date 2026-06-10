// Package notifier émet des événements vers le notification-service après une
// action (like, commentaire, réponse, mention, suppression). C'est de la
// communication serveur-à-serveur sur le réseau Docker : best-effort et
// fire-and-forget (un échec n'impacte JAMAIS l'action côté post-service).
//
// Le contrat JSON est aligné sur `models.Event` du notification-service.
package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"
)

// Types d'événement (alignés sur le notification-service).
const (
	TypeLike         = "like"
	TypeComment      = "comment"
	TypeReply        = "reply"
	TypeMention      = "mention"
	TypeRepost       = "repost"
	TypeQuote        = "quote"
	EventPostDeleted = "post_deleted"
	// EventPostPurgeWarning : préavis (≈1 mois) avant la purge définitive d'un
	// tweet masqué par la modération → notifie l'auteur (RecipientID).
	EventPostPurgeWarning = "post_purge_warning"
)

// Event — charge utile envoyée au notification-service.
type Event struct {
	Type           string   `json:"type"`
	ActorID        string   `json:"actor_id"`
	RecipientID    string   `json:"recipient_id,omitempty"`
	PostID         string   `json:"post_id,omitempty"`
	CommentID      string   `json:"comment_id,omitempty"`
	MentionHandles []string `json:"mention_handles,omitempty"`
	Retract        bool     `json:"retract,omitempty"`
}

// Notifier émet des événements (interface → service testable avec un faux/no-op).
type Notifier interface {
	Emit(ev Event)
}

// Noop ne fait rien : utilisé quand le notification-service n'est pas configuré
// (post-service reste autonome) ou dans les tests.
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
// pas casser un like / un commentaire.
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

// mentionRe capture les @handles (mêmes règles que les usernames :
// alphanumérique + underscore, 3 à 50 caractères). Le `@` doit être en début de
// texte ou précédé d'un caractère non-mot : on évite ainsi de capturer la partie
// domaine d'une adresse e-mail (`jean@exemple.com`). RE2 n'a pas de lookbehind →
// on consomme la frontière dans un groupe non capturant.
var mentionRe = regexp.MustCompile(`(?:^|[^\w@])@(\w{3,50})`)

// ParseMentions extrait les handles mentionnés dans un texte, dédupliqués
// (insensible à la casse pour la déduplication, handle d'origine conservé).
// Fonction PURE (testée).
func ParseMentions(content string) []string {
	matches := mentionRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		handle := m[1]
		key := toLowerASCII(handle)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, handle)
	}
	return out
}

// toLowerASCII abaisse la casse d'un handle ASCII (sans dépendance, suffisant
// pour [a-zA-Z0-9_]).
func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
