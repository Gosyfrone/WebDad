// Package eraser orchestre l'effacement RGPD des données applicatives d'un
// utilisateur à travers les services (profil, posts, messages, médias, graphe
// social). Utilisé par le balayage automatique des comptes bannis depuis 5 ans
// (auth-service mint un token admin et appelle les mêmes endpoints que l'admin
// depuis le front). auth-service efface SES propres identifiants à part.
package eraser

import (
	"context"
	"net/http"
	"time"
)

// Targets : URL de base de chaque service à purger (vide = service non
// configuré → étape ignorée, le service reste autonome).
type Targets struct {
	User    string
	Profil  string
	Post    string
	Message string
	Media   string
}

// Eraser appelle les endpoints de purge admin de chaque service.
type Eraser struct {
	targets Targets
	client  *http.Client
}

func New(t Targets) *Eraser {
	return &Eraser{targets: t, client: &http.Client{Timeout: 10 * time.Second}}
}

// Erase purge les données applicatives de `userID` sur tous les services
// configurés, avec un bearer admin. Renvoie la liste des étapes en échec (vide
// = succès complet). Un 404 est toléré (donnée déjà absente).
func (e *Eraser) Erase(ctx context.Context, bearer, userID string) []string {
	steps := []struct{ label, base, path string }{
		{"profil", e.targets.Profil, "/profils/" + userID},
		{"posts", e.targets.Post, "/posts/by-author/" + userID},
		{"messages", e.targets.Message, "/messages/users/" + userID},
		{"media", e.targets.Media, "/media/owners/" + userID},
		{"user", e.targets.User, "/users/" + userID + "/hard"},
	}
	var failed []string
	for _, s := range steps {
		if s.base == "" {
			continue // service non configuré → ignoré
		}
		if !e.deleteOK(ctx, bearer, s.base+s.path) {
			failed = append(failed, s.label)
		}
	}
	return failed
}

// deleteOK effectue un DELETE authentifié ; succès = 2xx ou 404 (idempotent).
func (e *Eraser) deleteOK(ctx context.Context, bearer, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := e.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode < 300 || resp.StatusCode == http.StatusNotFound
}
