package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health : GET /health — sonde de vivacité du service.
func Health(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
	}
}
