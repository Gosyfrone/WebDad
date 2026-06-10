package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/middleware"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/service"
)

// KeyHandler gère le registre des clés publiques d'identité (E2EE).
type KeyHandler struct {
	service *service.MessageService
}

func NewKeyHandler(svc *service.MessageService) *KeyHandler {
	return &KeyHandler{service: svc}
}

// PublishKey : PUT /messages/keys — publie/met à jour MA clé publique (l'id est
// dérivé du JWT, jamais du corps). La clé privée reste dans le navigateur.
// @Summary     Publier / mettre à jour sa clé publique E2EE
// @Tags        messages
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.PublishKeyRequest true "Clé publique (base64)"
// @Success     200 {object} map[string]string "user_id + public_key"
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Router      /messages/keys [put]
func (h *KeyHandler) PublishKey(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.PublishKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.service.PublishKey(c.Request.Context(), claims.UserID, req.PublicKey); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user_id": claims.UserID, "public_key": req.PublicKey}})
}

// GetKey : GET /messages/keys/:userId — clé publique d'un utilisateur (pour
// emballer la clé de contenu à son intention). 404 s'il n'a pas encore de clé.
// @Summary     Récupérer la clé publique d'un utilisateur
// @Tags        messages
// @Produce     json
// @Security    BearerAuth
// @Param       userId path string true "User ID"
// @Success     200 {object} map[string]string "public_key"
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /messages/keys/{userId} [get]
func (h *KeyHandler) GetKey(c *gin.Context) {
	key, err := h.service.GetKey(c.Request.Context(), c.Param("userId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": key})
}
