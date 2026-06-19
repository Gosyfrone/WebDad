// Package client porte les appels serveur-à-serveur SORTANTS du report-service
// vers les autres services, sur le réseau Docker, authentifiés par un secret
// partagé (en-tête X-Internal-Secret) — JAMAIS routés par la gateway (même
// pattern que l'émission d'événements de post-service vers notification-service).
package client

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"time"
)

// PostModerator masque/démasque un post dans le post-service au gré des
// décisions de modération. Interface → service testable avec un faux/no-op.
type PostModerator interface {
	// AutoHide masque un post atteint par le seuil de signalements (en attente
	// d'une décision du modérateur). Best-effort, non bloquant.
	AutoHide(postID string)
	// AutoUnhide rétablit la visibilité d'un post (entité jugée conforme par la
	// modération). Best-effort, non bloquant.
	AutoUnhide(postID string)
}

// NoopPostModerator ne fait rien : utilisé quand le post-service n'est pas
// configuré (report-service reste autonome) ou dans les tests.
type NoopPostModerator struct{}

func (NoopPostModerator) AutoHide(string)   {}
func (NoopPostModerator) AutoUnhide(string) {}

// HTTPPostClient appelle les endpoints internes du post-service.
type HTTPPostClient struct {
	baseURL string // ex. http://post-service:8084
	secret  string
	client  *http.Client
}

// NewPostClient construit le client. `baseURL` est la base du post-service.
func NewPostClient(baseURL, secret string) *HTTPPostClient {
	return &HTTPPostClient{
		baseURL: baseURL,
		secret:  secret,
		client:  &http.Client{Timeout: 4 * time.Second},
	}
}

func (c *HTTPPostClient) AutoHide(postID string)   { c.call(postID, "auto-hide") }
func (c *HTTPPostClient) AutoUnhide(postID string) { c.call(postID, "auto-unhide") }

// call poste en arrière-plan (fire-and-forget) sur l'endpoint interne du
// post-service. Un échec est journalisé mais n'impacte JAMAIS l'appelant :
// la visibilité est sécurité-sensible côté post-service (barrière serveur), donc
// on trace clairement les ratés sans casser le signalement / la décision.
func (c *HTTPPostClient) call(postID, action string) {
	if postID == "" {
		return
	}
	url := c.baseURL + "/internal/posts/" + postID + "/" + action
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(nil))
		if err != nil {
			slog.Error("post auto-modération : requête", "action", action, "post_id", postID, "error", err)
			return
		}
		req.Header.Set("X-Internal-Secret", c.secret)

		resp, err := c.client.Do(req)
		if err != nil {
			slog.Error("post auto-modération : envoi", "action", action, "post_id", postID, "error", err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode >= 300 {
			slog.Error("post auto-modération : réponse", "action", action, "post_id", postID, "status", resp.StatusCode)
		}
	}()
}
