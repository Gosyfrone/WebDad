package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/notification-service/internal/middleware"
	"github.com/webdad/notification-service/internal/realtime"
	"github.com/webdad/notification-service/internal/service"
)

// RegisterRoutes enregistre les routes du notification-service.
//
// Sous `/notifications` : l'API authentifiée du destinataire (lecture, lu/non-lu,
// WebSocket) — c'est le seul préfixe que l'API Gateway mappe vers ce service.
//
// Sous `/internal` : l'ingestion d'événements (POST /internal/events), protégée
// par un secret partagé et appelée en direct par post-service sur le réseau
// Docker (jamais routée par la gateway, jamais exposée au client).
func RegisterRoutes(
	r *gin.Engine,
	serviceName string,
	svc *service.NotificationService,
	hub *realtime.Hub,
	jwtSecret string,
	internalSecret string,
	allowedOrigins []string,
) {
	auth := middleware.JWTAuth(jwtSecret)

	r.GET("/health", Health(serviceName))

	notifH := NewNotificationHandler(svc)
	wsH := NewWSHandler(hub, jwtSecret, allowedOrigins)
	internalH := NewInternalHandler(svc, internalSecret)

	// Ingestion serveur-à-serveur (secret partagé, pas de JWT).
	r.POST("/internal/events", internalH.Events)

	notifications := r.Group("/notifications")
	{
		// Temps réel : token en query param (le navigateur n'autorise pas
		// d'en-tête sur un upgrade WS) → validation interne au handler.
		notifications.GET("/ws", wsH.Connect)

		notifications.GET("", auth, notifH.List)
		notifications.GET("/unread-count", auth, notifH.UnreadCount)
		notifications.POST("/read", auth, notifH.MarkAllRead)
		notifications.POST("/:id/read", auth, notifH.MarkRead)
	}
}
