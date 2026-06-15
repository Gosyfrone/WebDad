package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/logging"
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
// @Success     200 {object} models.AuthUser "Adresse vérifiée et session ouverte — data: {message, token, refresh_token, user}"
// @Failure     400 {object} map[string]string "Token invalide ou expiré (code: invalid_token)"
// @Failure     403 {object} map[string]string "Compte désactivé"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/verify-email/confirm [post]
func (h *Handler) ConfirmVerifyEmail(c *gin.Context) {
	var req models.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	token, refresh, user, err := h.auth.VerifyEmail(req.Token)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidToken):
			logging.FromGin(c).Warn("vérification e-mail échouée", "reason", "invalid_token")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "lien de vérification invalide ou expiré",
				"code":  "invalid_token",
			})
		case errors.Is(err, services.ErrUserInactive):
			logging.FromGin(c).Warn("vérification e-mail échouée", "reason", "user_inactive")
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			logging.FromGin(c).Error("vérification e-mail : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "vérification impossible"})
		}
		return
	}

	logging.FromGin(c).Info("e-mail vérifié", "user_id", user.ID)
	// Session émise (comme /login) : le BFF posera le cookie refresh et renverra
	// l'access token au client → l'utilisateur entre directement dans l'app.
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message":       "Adresse e-mail vérifiée.",
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
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
