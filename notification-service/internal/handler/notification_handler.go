package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/webdad/notification-service/internal/middleware"
	"github.com/webdad/notification-service/internal/service"
)

// NotificationHandler sert l'API authentifiée du destinataire (lecture + lu/non-lu).
type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

// List : GET /notifications?limit=&before=<id> — notifications de l'utilisateur
// courant, paginées par curseur (de la plus récente à la plus ancienne).
func (h *NotificationHandler) List(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	notifs, err := h.service.List(c.Request.Context(), claims.UserID, parseLimit(c), c.Query("before"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": notifs})
}

// UnreadCount : GET /notifications/unread-count — pour le badge.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	count, err := h.service.UnreadCount(c.Request.Context(), claims.UserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": count}})
}

// MarkAllRead : POST /notifications/read — marque tout comme lu.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if err := h.service.MarkAllRead(c.Request.Context(), claims.UserID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkRead : POST /notifications/:id/read — marque une notification comme lue.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "claims absents"})
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), claims.UserID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// parseLimit lit ?limit (défaut/borne appliqués côté service).
func parseLimit(c *gin.Context) int64 {
	n, err := strconv.ParseInt(c.Query("limit"), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
