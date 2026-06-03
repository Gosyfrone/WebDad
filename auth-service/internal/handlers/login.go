package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// Login : POST /auth/login — vérifie les credentials, retourne un JWT.
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	token, user, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connexion impossible"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token": token,
		"user":  models.NewAuthUser(user),
	}})
}
