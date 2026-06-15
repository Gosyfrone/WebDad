package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/client"
	"github.com/webdad/user-service/internal/config"
	"github.com/webdad/user-service/internal/db"
	"github.com/webdad/user-service/internal/logging"
	"github.com/webdad/user-service/internal/repository"
	"github.com/webdad/user-service/internal/router"
	"github.com/webdad/user-service/internal/service"
)

const serviceName = "user-service"

func main() {
	logging.Setup(serviceName)

	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connexion DB", "error", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	if err := db.EnsureSchema(conn); err != nil {
		slog.Error("schéma", "error", err)
		os.Exit(1)
	}

	opts := []service.Option{
		service.WithProfilClient(client.NewProfilClient(cfg.ProfilServiceURL)),
	}
	if cfg.NotificationServiceURL != "" {
		opts = append(opts, service.WithNotificationClient(client.NewNotificationClient(cfg.NotificationServiceURL, cfg.InternalSecret)))
	}
	users := service.New(repository.New(conn), cfg.UsernameCooldown, opts...)
	r := router.New(users, cfg.JWTSecret)

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
