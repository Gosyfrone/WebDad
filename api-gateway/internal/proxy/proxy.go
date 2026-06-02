// Package proxy construit les reverse proxies vers les services backend.
package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// New retourne un handler Gin qui transmet la requête entrante au service
// cible (méthode, chemin, corps et en-têtes inchangés) et renvoie sa réponse
// telle quelle au client. Le préfixe de route est conservé : une requête
// `/auth/register` arrive au service sur `/auth/register`.
func New(target string) (gin.HandlerFunc, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	// Réponse homogène ({"error": ...}) si le service est injoignable, plutôt
	// que la page d'erreur 502 par défaut du reverse proxy.
	rp.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, e error) {
		log.Printf("[gateway] service injoignable (%s) : %v", target, e)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"service indisponible"}`))
	}

	return func(c *gin.Context) {
		rp.ServeHTTP(c.Writer, c.Request)
	}, nil
}
