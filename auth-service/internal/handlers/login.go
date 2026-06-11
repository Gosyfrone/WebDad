package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// Login : POST /auth/login — vérifie les credentials, retourne un JWT.
// @Summary     Se connecter
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.LoginRequest true "Credentials"
// @Success     200 {object} models.AuthUser "Connexion réussie — data: {token, refresh_token, user}"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     401 {object} map[string]string "Credentials invalides"
// @Failure     403 {object} map[string]string "Compte désactivé ou e-mail non vérifié (code: email_not_verified)"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	token, refresh, user, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrEmailNotVerified):
			// Code machine pour que le front propose le renvoi du mail de vérif.
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "code": "email_not_verified"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connexion impossible"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
}
