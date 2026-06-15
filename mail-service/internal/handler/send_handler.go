package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/mail-service/internal/mailer"
	"github.com/webdad/mail-service/internal/models"
)

// SendHandler traite l'envoi d'e-mails demandé par les autres services
// (auth-service) sur le réseau Docker. Protégé par un secret partagé
// (en-tête X-Internal-Secret), JAMAIS routé par la gateway ni exposé au
// client : communication serveur-à-serveur (même principe que
// notification-service /internal/events).
type SendHandler struct {
	mailer mailer.Mailer
	secret string
}

func NewSendHandler(m mailer.Mailer, secret string) *SendHandler {
	return &SendHandler{mailer: m, secret: secret}
}

// Send : POST /internal/send — envoie un e-mail.
// Répond 202 uniquement quand le transport a accepté le message.
func (h *SendHandler) Send(c *gin.Context) {
	if c.GetHeader("X-Internal-Secret") != h.secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "secret interne invalide"})
		return
	}

	var req models.SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}
	if req.HTML == "" && req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "html ou text requis"})
		return
	}

	if err := h.mailer.Send(mailer.Message{
		To:      req.To,
		Subject: req.Subject,
		HTML:    req.HTML,
		Text:    req.Text,
	}); err != nil {
		slog.Error("envoi e-mail échoué", "subject", req.Subject, "error", err)
		if errors.Is(err, mailer.ErrDeliveryUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "transport e-mail non configuré",
				"code":  "delivery_unavailable",
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "envoi impossible"})
		return
	}
	slog.Info("e-mail envoyé", "subject", req.Subject)
	c.Status(http.StatusAccepted)
}
