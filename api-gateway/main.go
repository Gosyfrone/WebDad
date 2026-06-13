package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/webdad/api-gateway/internal/config"
	"github.com/webdad/api-gateway/internal/logging"
	"github.com/webdad/api-gateway/internal/router"
)

const serviceName = "api-gateway"

func main() {
	logging.Setup(serviceName)

	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	r, err := router.New(cfg)
	if err != nil {
		slog.Error("configuration du routeur", "error", err)
		os.Exit(1)
	}

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
