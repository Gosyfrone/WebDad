package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

// Register : POST /auth/register — crée un compte (role=user).
// @Summary     Créer un compte
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body models.RegisterRequest true "Email + mot de passe (min 8 chars)"
// @Success     201 {object} models.AuthUser "Compte créé — data: {token, refresh_token, user}"
// @Failure     400 {object} map[string]string "Payload invalide"
// @Failure     409 {object} map[string]string "Email déjà utilisé"
// @Failure     500 {object} map[string]string "Erreur interne"
// @Router      /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	token, refresh, user, err := h.auth.Register(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "création du compte impossible"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"token":         token,
		"refresh_token": refresh,
		"user":          models.NewAuthUser(user),
	}})
}
