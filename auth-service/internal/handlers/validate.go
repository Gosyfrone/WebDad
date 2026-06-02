package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/services"
)

// Validate : GET /auth/validate — destinée à l'API Gateway pour vérifier
// un token avant de relayer vers les autres services. Le middleware JWT a
// déjà validé le token et posé les claims dans le contexte ; on les renvoie.
func (h *Handler) Validate(c *gin.Context) {
	val, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	claims, ok := val.(*services.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "claims illisibles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	}})
}
