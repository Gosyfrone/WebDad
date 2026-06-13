package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/webdad/mail-service/internal/config"
	"github.com/webdad/mail-service/internal/handler"
	"github.com/webdad/mail-service/internal/logging"
	"github.com/webdad/mail-service/internal/mailer"
	"github.com/webdad/mail-service/internal/middleware"
)

const serviceName = "mail-service"

func main() {
	logging.Setup(serviceName)

	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	// Le mailer choisit son transport au démarrage : SMTP si configuré,
	// sinon repli console (les e-mails sont loggés, pratique en dev).
	m := mailer.New(cfg)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	handler.RegisterRoutes(r, serviceName, m, cfg.InternalSecret)

	slog.Info("en écoute", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("échec du démarrage", "error", err)
		os.Exit(1)
	}
}

// ginMode borne la valeur de GIN_MODE aux modes connus (défaut : debug).
func ginMode(mode string) string {
	switch mode {
	case gin.ReleaseMode, gin.TestMode:
		return mode
	default:
		return gin.DebugMode
	}
}
