package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// ForgotPassword : POST /auth/password/forgot — déclenche l'envoi du mail de
// réinitialisation. ANTI-ÉNUMÉRATION : répond TOUJOURS 200 avec un message
// générique, que le compte existe, soit actif, ou non.
// @Summary     Demander une réinitialisation de mot de passe
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.ForgotPasswordRequest true "Adresse e-mail"
// @Success     200 {object} map[string]string "Réponse générique — data: {message}"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Router      /auth/password/forgot [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	// Toujours nil (anti-énumération) : on ignore volontairement le retour.
	_ = h.auth.ForgotPassword(req.Email)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message": "Si un compte correspond à cette adresse, un e-mail de réinitialisation vient d'être envoyé.",
	}})
}

// ResetPassword : POST /auth/password/reset — consomme le token reçu par e-mail
// et remplace le mot de passe. Révoque toutes les sessions ouvertes.
// @Summary     Réinitialiser le mot de passe
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.ResetPasswordRequest true "Token de reset (issu du lien e-mail) + nouveau mot de passe"
// @Success     200 {object} map[string]string "Mot de passe réinitialisé — data: {message}"
// @Failure     400 {object} map[string]string "Token invalide ou expiré (code: invalid_token) / payload invalide"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/password/reset [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.auth.ResetPassword(req.Token, req.NewPassword); err != nil {
		if errors.Is(err, services.ErrInvalidToken) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "lien de réinitialisation invalide ou expiré",
				"code":  "invalid_token",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "réinitialisation impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message": "Mot de passe réinitialisé. Tu peux te connecter avec ton nouveau mot de passe.",
	}})
}
