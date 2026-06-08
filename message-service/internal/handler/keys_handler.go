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
func (h *KeyHandler) GetKey(c *gin.Context) {
	key, err := h.service.GetKey(c.Request.Context(), c.Param("userId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": key})
}
