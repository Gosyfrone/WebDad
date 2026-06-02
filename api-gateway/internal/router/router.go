// Package router enregistre les routes du gateway (health + reverse proxy).
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/api-gateway/internal/config"
	"github.com/webdad/api-gateway/internal/middleware"
	"github.com/webdad/api-gateway/internal/proxy"
)

const serviceName = "api-gateway"

// New construit le routeur : CORS, /health, puis un reverse proxy par service.
func New(cfg *config.Config) (*gin.Engine, error) {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
	})

	// Une route attrape-tout par préfixe → reverse proxy vers le service.
	// (Routes publiques pour l'instant ; le middleware JWT viendra protéger
	// les préfixes concernés quand on branchera le login.)
	for prefix, target := range cfg.Services {
		h, err := proxy.New(target)
		if err != nil {
			return nil, err
		}
		r.Any(prefix+"/*path", h)
	}

	return r, nil
}
