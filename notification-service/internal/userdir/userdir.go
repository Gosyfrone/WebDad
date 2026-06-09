// Package userdir résout un handle (@pseudo) en identifiant utilisateur, en
// interrogeant le user-service. C'est le SEUL couplage du notification-service
// vers un autre service, nécessaire car seul user-service connaît les handles
// (les mentions sont saisies sous forme de @pseudo, jamais d'id côté front).
package userdir

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Resolver expose la résolution handle → id (interface → testable par un faux).
type Resolver interface {
	// ResolveHandle renvoie l'id de l'utilisateur portant ce handle. `ok` est
	// false si le handle est inconnu (404) ; une erreur réseau est remontée.
	ResolveHandle(ctx context.Context, handle string) (id string, ok bool, err error)
}

// HTTPResolver interroge `GET {base}/users/by-username/:username`.
type HTTPResolver struct {
	base   string
	client *http.Client
}

// New construit un résolveur HTTP vers le user-service.
func New(base string) *HTTPResolver {
	return &HTTPResolver{
		base:   base,
		client: &http.Client{Timeout: 4 * time.Second},
	}
}

func (r *HTTPResolver) ResolveHandle(ctx context.Context, handle string) (string, bool, error) {
	endpoint := fmt.Sprintf("%s/users/by-username/%s", r.base, url.PathEscape(handle))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("user-service a répondu %d", resp.StatusCode)
	}

	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", false, err
	}
	if body.Data.ID == "" {
		return "", false, nil
	}
	return body.Data.ID, true, nil
}
