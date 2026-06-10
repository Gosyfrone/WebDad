package handler

import (
	"time"

	"github.com/gin-gonic/gin"
)

// startedAt : instant d'init du package (≈ démarrage du process). Sert à
// exposer l'uptime du service dans /health (consommé par le monitoring admin).
var startedAt = time.Now()

// Health : sonde de liveness (healthcheck Docker / CI).
func Health(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":         "ok",
			"service":        service,
			"uptime_seconds": int(time.Since(startedAt).Seconds()),
		})
	}
}
