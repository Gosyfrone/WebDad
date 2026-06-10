package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// ConfirmVerifyEmail : POST /auth/verify-email/confirm — consomme le token reçu
// par e-mail et marque l'adresse vérifiée.
// @Summary     Confirmer la vérification d'e-mail
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.VerifyEmailRequest true "Token de vérification (issu du lien e-mail)"
// @Success     200 {object} map[string]string "Adresse vérifiée — data: {message}"
// @Failure     400 {object} map[string]string "Token invalide ou expiré (code: invalid_token)"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/verify-email/confirm [post]
func (h *Handler) ConfirmVerifyEmail(c *gin.Context) {
	var req models.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.auth.VerifyEmail(req.Token); err != nil {
		if errors.Is(err, services.ErrInvalidToken) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "lien de vérification invalide ou expiré",
				"code":  "invalid_token",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vérification impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "Adresse e-mail vérifiée."}})
}

// RequestVerifyEmail : POST /auth/verify-email/request — (ré)envoie le mail de
// vérification. ANTI-ÉNUMÉRATION : répond TOUJOURS 200 avec un message
// générique, que le compte existe, soit déjà vérifié, ou non.
// @Summary     Renvoyer le mail de vérification
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.RequestVerifyRequest true "Adresse e-mail"
// @Success     200 {object} map[string]string "Réponse générique — data: {message}"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Router      /auth/verify-email/request [post]
func (h *Handler) RequestVerifyEmail(c *gin.Context) {
	var req models.RequestVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	// Toujours nil (anti-énumération) : on ignore volontairement le retour.
	_ = h.auth.ResendVerification(req.Email)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message": "Si un compte non vérifié correspond à cette adresse, un e-mail de vérification vient d'être envoyé.",
	}})
}
