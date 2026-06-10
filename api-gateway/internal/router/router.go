// Package router enregistre les routes du gateway (health + reverse proxy).
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/api-gateway/internal/config"
	"github.com/webdad/api-gateway/internal/middleware"
	"github.com/webdad/api-gateway/internal/monitoring"
	"github.com/webdad/api-gateway/internal/proxy"
)

const serviceName = "api-gateway"

// New construit le routeur : CORS, /health, puis un reverse proxy par service.
func New(cfg *config.Config) (*gin.Engine, error) {
	r := gin.Default()

	// On NE redirige PAS sur le slash final : la gateway transmet le chemin tel
	// quel au service. Sans ça, un `POST /users` (endpoint collection, sans
	// sous-chemin) serait redirigé en 307 vers `/users/` que le service ne
	// connaît pas → boucle de redirections. On gère donc explicitement le
	// préfixe nu ET les sous-chemins ci-dessous.
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
	})

	// Monitoring infra (console admin) : agrège la santé de tous les services.
	// Servi DIRECTEMENT par le gateway (pas un proxy) et réservé aux admins.
	// Le préfixe `/admin` n'est pas une cible de proxy → aucun conflit.
	r.GET("/admin/monitoring", middleware.AdminJWT(cfg.JWTSecret), monitoring.Handler(cfg.Services))

	// Reverse proxy par préfixe vers le service cible. Deux routes par service :
	//   - le préfixe nu (`/users`)        → endpoints collection (list, create)
	//   - le préfixe + sous-chemin (`/users/*path`) → ressources (`/users/me`, …)
	// (Routes publiques pour l'instant ; le middleware JWT viendra protéger les
	// préfixes concernés quand on branchera le login.)
	for prefix, target := range cfg.Services {
		h, err := proxy.New(target)
		if err != nil {
			return nil, err
		}
		r.Any(prefix, h)
		r.Any(prefix+"/*path", h)
	}

	return r, nil
}
