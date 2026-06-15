package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/logging"
	"github.com/webdad/auth-service/internal/middleware"
	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// RequestEmailChange démarre un changement d'adresse pour le compte connecté.
// @Summary     Demander un changement d'adresse e-mail
// @Tags        auth
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.ChangeEmailRequest true "Nouvelle adresse e-mail"
// @Success     200 {object} map[string]string "Lien de confirmation envoyé à la nouvelle adresse"
// @Failure     400 {object} map[string]string "Payload invalide, adresse identique ou déjà utilisée"
// @Failure     401 {object} map[string]string "Non authentifié"
// @Failure     403 {object} map[string]string "Compte désactivé"
// @Failure     404 {object} map[string]string "Compte introuvable"
// @Failure     503 {object} map[string]string "Service d'envoi d'e-mail indisponible"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/email/change/request [post]
func (h *Handler) RequestEmailChange(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	var req models.ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	err := h.auth.RequestEmailChange(claims.UserID, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailTaken):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "email_taken"})
		case errors.Is(err, services.ErrSameEmail):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "same_email"})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrEmailDelivery):
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "l'e-mail de confirmation n'a pas pu être envoyé",
				"code":  "email_delivery_failed",
			})
		default:
			logging.FromGin(c).Error("demande changement e-mail : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "changement d'adresse impossible"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "Un lien de confirmation a été envoyé à la nouvelle adresse."}})
}

// ConfirmEmailChange confirme la nouvelle adresse et ouvre une nouvelle session.
// @Summary     Confirmer un changement d'adresse e-mail
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.ConfirmEmailChangeRequest true "Jeton reçu à la nouvelle adresse"
// @Success     200 {object} models.AuthUser "Adresse modifiée et session ouverte — data: {token, refresh_token, user}"
// @Failure     400 {object} map[string]string "Jeton invalide/expiré ou adresse devenue indisponible"
// @Failure     403 {object} map[string]string "Compte désactivé"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/email/change/confirm [post]
func (h *Handler) ConfirmEmailChange(c *gin.Context) {
	var req models.ConfirmEmailChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	token, refresh, user, err := h.auth.ConfirmEmailChange(req.Token)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidToken):
			c.JSON(http.StatusBadRequest, gin.H{"error": "lien invalide ou expiré", "code": "invalid_token"})
		case errors.Is(err, services.ErrEmailTaken):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "email_taken"})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			logging.FromGin(c).Error("confirmation changement e-mail : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "confirmation impossible"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message": "Adresse e-mail modifiée.", "token": token,
		"refresh_token": refresh, "user": models.NewAuthUser(user),
	}})
}
