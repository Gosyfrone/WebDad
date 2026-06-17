package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/service"
)

// InternalHandler reçoit les commandes d'auto-modération émises par le
// report-service sur le réseau Docker. Protégé par un secret partagé (en-tête
// X-Internal-Secret), JAMAIS exposé au client ni routé par la gateway : c'est de
// la communication serveur-à-serveur (même pattern que l'ingestion d'événements
// du notification-service).
type InternalHandler struct {
	service *service.PostService
	secret  string
}

func NewInternalHandler(svc *service.PostService, secret string) *InternalHandler {
	return &InternalHandler{service: svc, secret: secret}
}

// authorized refuse l'appel si le secret interne ne correspond pas. Renvoie
// false après avoir écrit la réponse 401.
func (h *InternalHandler) authorized(c *gin.Context) bool {
	if h.secret == "" || c.GetHeader("X-Internal-Secret") != h.secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "secret interne invalide"})
		return false
	}
	return true
}

// AutoHide : POST /internal/posts/:id/auto-hide — masque un post trop signalé.
func (h *InternalHandler) AutoHide(c *gin.Context) {
	if !h.authorized(c) {
		return
	}
	if _, err := h.service.AutoHide(c.Request.Context(), c.Param("id")); err != nil {
		slog.Error("auto-hide interne", "post_id", c.Param("id"), "error", err)
		respondPostError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AutoUnhide : POST /internal/posts/:id/auto-unhide — rétablit un post (conforme).
func (h *InternalHandler) AutoUnhide(c *gin.Context) {
	if !h.authorized(c) {
		return
	}
	if _, err := h.service.AutoUnhide(c.Request.Context(), c.Param("id")); err != nil {
		slog.Error("auto-unhide interne", "post_id", c.Param("id"), "error", err)
		respondPostError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
