package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/realtime"
)

// WSHandler gère la poignée de main WebSocket du fil temps réel.
type WSHandler struct {
	hub      *realtime.Hub
	secret   string
	upgrader websocket.Upgrader
}

// NewWSHandler construit le handler. L'upgrader n'accepte que les origines
// autorisées (mêmes que la CORS) ; une requête sans en-tête Origin (client non
// navigateur, ex. tests) est acceptée.
func NewWSHandler(hub *realtime.Hub, secret string, allowedOrigins []string) *WSHandler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return &WSHandler{
		hub:    hub,
		secret: secret,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowed[origin]
			},
		},
	}
}

// Connect : GET /posts/ws?access_token=<jwt> — ouvre la connexion temps réel du
// fil. Le token transite en query param car le navigateur n'autorise pas
// d'en-tête Authorization sur une poignée de main WebSocket. Le client reçoit
// ensuite un « ping » à chaque nouveau post public racine (id + auteur).
// @Summary     Connexion WebSocket temps réel (fil)
// @Tags        posts
// @Param       access_token query string true "JWT access token"
// @Success     101 {string} string "Upgrade WebSocket"
// @Router      /posts/ws [get]
func (h *WSHandler) Connect(c *gin.Context) {
	claims, err := middleware.ParseToken(c.Query("access_token"), h.secret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token invalide ou expiré"})
		return
	}

	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// L'upgrader a déjà écrit la réponse d'erreur.
		return
	}

	slog.Debug("ws fil connecté", "user_id", claims.UserID)
	h.hub.Register(ws)
}
