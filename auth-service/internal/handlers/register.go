package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// Register : POST /auth/register — crée un compte (role=user).
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	token, user, err := h.auth.Register(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "création du compte impossible"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"token": token,
		"user":  models.NewAuthUser(user),
	}})
}
