package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health : sonde de liveness (healthcheck Docker / CI).
func Health(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": service,
		})
	}
}
