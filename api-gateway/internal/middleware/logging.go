package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestID lit X-Request-Id ou génère un id (16 octets hex), le pose dans le
// contexte Gin, l'en-tête de réponse ET l'en-tête de requête sortante pour que
// le reverse proxy le transmette aux services backend (corrélation inter-services).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Set("request_id", id)
		c.Header("X-Request-Id", id)
		c.Request.Header.Set("X-Request-Id", id)
		c.Next()
	}
}

// RequestLogger enregistre une ligne slog par requête : method, path (jamais
// la query string), status, latency_ms, client_ip, request_id, upstream déduit
// du premier segment du chemin. Niveau error ≥500, warn ≥400, info sinon.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"request_id", c.GetString("request_id"),
			"upstream", upstreamFromPath(c.Request.URL.Path),
		}

		switch {
		case status >= http.StatusInternalServerError:
			slog.Error("request", attrs...)
		case status >= http.StatusBadRequest:
			slog.Warn("request", attrs...)
		default:
			slog.Info("request", attrs...)
		}
	}
}

// Recovery capture les panics Gin, les logue avec slog et répond 500.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, rec any) {
		slog.Error("panic récupéré", "error", rec, "request_id", c.GetString("request_id"))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
	})
}

// upstreamFromPath extrait le premier segment du chemin ("/auth/register" → "auth").
func upstreamFromPath(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	if idx := strings.IndexByte(trimmed, '/'); idx != -1 {
		return trimmed[:idx]
	}
	return trimmed
}
