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
			logging.FromGin(c).Warn("reset mot de passe refusé", "reason", "invalid_token")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "lien de réinitialisation invalide ou expiré",
				"code":  "invalid_token",
			})
			return
		}
		logging.FromGin(c).Error("reset mot de passe : erreur inattendue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "réinitialisation impossible"})
		return
	}

	logging.FromGin(c).Info("mot de passe réinitialisé")
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message": "Mot de passe réinitialisé. Tu peux te connecter avec ton nouveau mot de passe.",
	}})
}

// ChangePassword : POST /auth/password/change — change le mot de passe de
// l'utilisateur authentifié après vérification du mot de passe actuel. Sert au
// changement volontaire ET au changement imposé (mot de passe temporaire posé
// par un admin) : lève le drapeau must_change_password et ré-émet une paire de
// tokens (le nouveau JWT ne porte plus le drapeau). Les autres sessions sont révoquées.
// @Summary     Changer son mot de passe
// @Tags        auth
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.ChangePasswordRequest true "Mot de passe actuel + nouveau mot de passe (min 8 chars)"
// @Success     200 {object} models.AuthUser "Mot de passe changé — data: {token, refresh_token, user}"
// @Failure     400 {object} map[string]string "Payload invalide ou mot de passe actuel invalide (code: invalid_current_password)"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     404 {object} map[string]string "Compte introuvable"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/password/change [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	token, refresh, user, err := h.auth.ChangePassword(claims.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCurrentPassword):
			logging.FromGin(c).Warn("changement mot de passe refusé", "reason", "invalid_current_password")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
				"code":  "invalid_current_password",
			})
		case errors.Is(err, services.ErrUserNotFound):
			logging.FromGin(c).Warn("changement mot de passe refusé", "reason", "user_not_found")
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			logging.FromGin(c).Error("changement mot de passe : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "changement de mot de passe impossible"})
		}
		return
	}

	logging.FromGin(c).Info("mot de passe changé", "user_id", user.ID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
}
