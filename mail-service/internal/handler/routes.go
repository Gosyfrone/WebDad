package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/mail-service/internal/mailer"
)

// RegisterRoutes enregistre les routes du mail-service.
//
// `/health` sert le healthcheck Docker / CI.
//
// Sous `/internal` : l'envoi d'e-mails (POST /internal/send), protégé par un
// secret partagé et appelé en direct par auth-service sur le réseau Docker
// (jamais routé par la gateway, jamais exposé au client).
func RegisterRoutes(r *gin.Engine, serviceName string, m mailer.Mailer, internalSecret string) {
	r.GET("/health", Health(serviceName))

	sendH := NewSendHandler(m, internalSecret)
	r.POST("/internal/send", sendH.Send)
}
