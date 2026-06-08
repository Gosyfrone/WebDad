package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/middleware"
	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/service"
)

// ConversationHandler : conversations + messages (le hub sert la diffusion WS).
type ConversationHandler struct {
	service *service.MessageService
	hub     *realtime.Hub
}

func NewConversationHandler(svc *service.MessageService, hub *realtime.Hub) *ConversationHandler {
	return &ConversationHandler{service: svc, hub: hub}
}

// CreateConversation : POST /messages/conversations — crée (ou retrouve) un DM.
// Le client a généré la clé de contenu et fournit les enveloppes par membre.
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.CreateDMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	view, err := h.service.CreateDM(c.Request.Context(), claims.UserID, req.PeerID, req.Envelopes)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": view})
}

// ListConversations : GET /messages/conversations — mes conversations (avec mon
// enveloppe + mon rôle), triées par activité.
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	views, err := h.service.ListConversations(c.Request.Context(), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

// GetConversation : GET /messages/conversations/:id — détail (membre requis).
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	view, err := h.service.GetConversation(c.Request.Context(), c.Param("id"), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// ListMessages : GET /messages/conversations/:id/messages — historique chiffré
// (membre requis). ?before=<messageId> pagine vers le haut ; ?limit borne la page.
func (h *ConversationHandler) ListMessages(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	msgs, err := h.service.ListMessages(
		c.Request.Context(), c.Param("id"), claims.UserID, pageLimit(c), c.Query("before"),
	)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": msgs})
}

// SendMessage : POST /messages/conversations/:id/messages — poste un message
// DÉJÀ chiffré (membre + droit d'écriture requis), puis le diffuse en temps réel
// aux membres connectés.
func (h *ConversationHandler) SendMessage(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}

	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	msg, memberIDs, err := h.service.SendMessage(
		c.Request.Context(), c.Param("id"), claims.UserID, req.Ciphertext, req.Nonce,
	)
	if err != nil {
		respondError(c, err)
		return
	}

	// Diffusion temps réel (le payload reste chiffré ; le serveur ne lit rien).
	h.hub.Publish(memberIDs, gin.H{"type": "message", "data": msg})

	c.JSON(http.StatusCreated, gin.H{"data": msg})
}

// pageLimit lit ?limit (défaut/borne appliqués côté service).
func pageLimit(c *gin.Context) int64 {
	n, err := strconv.ParseInt(c.Query("limit"), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
