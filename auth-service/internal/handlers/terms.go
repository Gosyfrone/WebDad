package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/logging"
	"github.com/webdad/auth-service/internal/middleware"
	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// AcceptTerms : POST /auth/terms/accept — enregistre l'acceptation par
// l'utilisateur authentifié de la version EN VIGUEUR des CGU et ré-émet une
// paire de tokens. Le nouveau JWT porte terms_accepted=true → la modale
// d'acceptation bloquante du front disparaît sans reconnexion.
// @Summary     Accepter les CGU en vigueur
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} models.AuthUser "CGU acceptées — data: {token, refresh_token, user}"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     404 {object} map[string]string "Compte introuvable"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/terms/accept [post]
func (h *Handler) AcceptTerms(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	token, refresh, user, err := h.auth.AcceptTerms(claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			logging.FromGin(c).Warn("acceptation CGU refusée", "reason", "user_not_found")
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logging.FromGin(c).Error("acceptation CGU : erreur inattendue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "acceptation des CGU impossible"})
		return
	}

	logging.FromGin(c).Info("CGU acceptées", "user_id", user.ID, "version", models.CurrentTermsVersion)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
}
