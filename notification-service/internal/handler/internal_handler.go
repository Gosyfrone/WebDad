package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/notification-service/internal/models"
	"github.com/webdad/notification-service/internal/service"
)

// InternalHandler reçoit les événements émis par les autres services (post-service)
// sur le réseau Docker. Protégé par un secret partagé (en-tête X-Internal-Secret),
// JAMAIS exposé au client ni routé par la gateway : c'est de la communication
// serveur-à-serveur (cf. la doc des décisions).
type InternalHandler struct {
	service *service.NotificationService
	secret  string
}

func NewInternalHandler(svc *service.NotificationService, secret string) *InternalHandler {
	return &InternalHandler{service: svc, secret: secret}
}

// Events : POST /internal/events — ingère un événement et l'agrège.
// Répond 202 (accepté) sans attendre la diffusion : l'émetteur est best-effort.
func (h *InternalHandler) Events(c *gin.Context) {
	if c.GetHeader("X-Internal-Secret") != h.secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "secret interne invalide"})
		return
	}

	var ev models.Event
	if err := c.ShouldBindJSON(&ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.service.HandleEvent(c.Request.Context(), ev); err != nil {
		slog.Error("traitement événement échoué", "event_type", ev.Type, "actor_id", ev.ActorID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "traitement impossible"})
		return
	}
	slog.Debug("événement traité", "event_type", ev.Type, "actor_id", ev.ActorID, "recipient_id", ev.RecipientID)
	c.Status(http.StatusAccepted)
}
