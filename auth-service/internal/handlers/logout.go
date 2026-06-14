package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/logging"
	"github.com/webdad/auth-service/internal/models"
)

// Logout : POST /auth/logout — révoque le refresh token (corps JSON, transmis
// par le BFF Next depuis le cookie httpOnly). Best-effort et idempotent :
// renvoie 200 même si le token est absent ou déjà révoqué (le BFF efface le
// cookie de son côté dans tous les cas).
// @Summary     Se déconnecter (best-effort)
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.RefreshRequest false "Refresh token (optionnel)"
// @Success     200 {object} map[string]string "Déconnecté"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	_ = c.ShouldBindJSON(&req) // corps absent toléré (déconnexion best-effort)

	if err := h.auth.Logout(req.RefreshToken); err != nil {
		logging.FromGin(c).Error("logout : révocation refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "déconnexion impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "déconnecté"}})
}
