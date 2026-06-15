// Package logging configure le logger structuré slog du service.
package logging

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
)

// Setup initialise slog.SetDefault avec un handler JSON (release) ou texte
// (debug/test). Niveau depuis LOG_LEVEL (debug|info|warn|error, défaut info).
// L'attribut "service" est ajouté à toutes les lignes.
func Setup(service string) {
	opts := &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))}
	var handler slog.Handler
	if os.Getenv("GIN_MODE") == "release" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler).With(slog.String("service", service)))
}

// FromGin retourne un logger enrichi avec request_id et user_id (si présent)
// pour corrélation automatique dans les handlers.
func FromGin(c *gin.Context) *slog.Logger {
	l := slog.Default().With("request_id", c.GetString("request_id"))
	if uid := c.GetString("user_id"); uid != "" {
		l = l.With("user_id", uid)
	}
	return l
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
