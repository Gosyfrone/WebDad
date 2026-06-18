package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/logging"
	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// Login : POST /auth/login — vérifie les credentials par email ou user_id, retourne un JWT.
// @Summary     Se connecter
// @Description Accepte `email` pour le login classique, ou `user_id` quand le BFF a résolu un username via user-service. `password` est toujours requis.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.LoginRequest true "Credentials (email ou user_id + password)"
// @Success     200 {object} models.AuthUser "Connexion réussie — data: {token, refresh_token, user} ; si MFA active : data: {mfa_required: true, challenge} (aucun token, appeler /auth/mfa/verify)"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     401 {object} map[string]string "Credentials invalides"
// @Failure     403 {object} map[string]string "Compte désactivé ou e-mail non vérifié (code: email_not_verified)"
// @Failure     409 {object} map[string]string "Compte sans mot de passe (connexion via Google)"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	email := strings.TrimSpace(req.Email)
	userID := strings.TrimSpace(req.UserID)
	if email == "" && userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email ou user_id requis"})
		return
	}

	var (
		outcome *services.LoginOutcome
		err     error
	)
	if userID != "" {
		outcome, err = h.auth.LoginByUserID(userID, req.Password)
	} else {
		outcome, err = h.auth.Login(email, req.Password)
	}
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			logging.FromGin(c).Warn("login échoué", "reason", "invalid_credentials")
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrNoLocalPassword):
			logging.FromGin(c).Warn("login échoué", "reason", "no_local_password")
			// 409 : conflit de méthode d'auth (le compte passe par un provider).
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrUserInactive):
			logging.FromGin(c).Warn("login échoué", "reason", "user_inactive")
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrEmailNotVerified):
			logging.FromGin(c).Warn("login échoué", "reason", "email_not_verified")
			// Code machine pour que le front propose le renvoi du mail de vérif.
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "code": "email_not_verified"})
		default:
			logging.FromGin(c).Error("login : erreur inattendue", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connexion impossible"})
		}
		return
	}

	// MFA active : mot de passe OK mais aucun JWT émis. On renvoie un challenge
	// court (5 min) ; le front affiche l'écran de code et appelle /auth/mfa/verify.
	if outcome.MFARequired {
		logging.FromGin(c).Info("login : second facteur requis", "user_id", outcome.User.ID)
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"mfa_required": true,
			"challenge":    outcome.Challenge,
		}})
		return
	}

	logging.FromGin(c).Info("connexion réussie", "user_id", outcome.User.ID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         outcome.Token,
		"refresh_token": outcome.Refresh,
		"user":          models.NewAuthUser(outcome.User),
	}})
}
