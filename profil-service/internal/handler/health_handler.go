package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// startedAt : instant d'init du package (≈ démarrage du process). Sert à
// exposer l'uptime du service dans /health (consommé par le monitoring admin).
var startedAt = time.Now()

// Health : GET /health — sonde de vivacité du service.
func Health(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"service":        serviceName,
			"uptime_seconds": int(time.Since(startedAt).Seconds()),
		})
	}
}
