package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// Refresh : POST /auth/refresh — échange un refresh token valide (corps JSON,
// transmis par le BFF Next depuis le cookie httpOnly) contre une NOUVELLE paire
// access + refresh (rotation). 401 si le token est absent/invalide/expiré : le
// front efface sa session et redirige vers /login.
// @Summary     Rafraîchir les tokens
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.RefreshRequest true "Refresh token"
// @Success     200 {object} models.AuthUser "Nouvelle paire tokens — data: {token, refresh_token, user}"
// @Failure     401 {object} map[string]string "Token absent/invalide/expiré"
// @Failure     403 {object} map[string]string "Compte désactivé"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token manquant"})
		return
	}

	token, refresh, user, err := h.auth.Refresh(req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRefreshToken):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rafraîchissement impossible"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
}
