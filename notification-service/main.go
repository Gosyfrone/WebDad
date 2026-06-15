package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/notification-service/internal/config"
	"github.com/webdad/notification-service/internal/database"
	"github.com/webdad/notification-service/internal/handler"
	"github.com/webdad/notification-service/internal/logging"
	"github.com/webdad/notification-service/internal/middleware"
	"github.com/webdad/notification-service/internal/realtime"
	"github.com/webdad/notification-service/internal/repository"
	"github.com/webdad/notification-service/internal/service"
	"github.com/webdad/notification-service/internal/userdir"
)

const serviceName = "notification-service"

func main() {
	logging.Setup(serviceName)

	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	client, err := database.ConnectMongo(cfg.MongoURI)
	if err != nil {
		slog.Error("connexion Mongo", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()

	db := client.Database(cfg.MongoDB)

	// Le service applique son propre schéma (collection + validateur + index)
	// au démarrage, de façon idempotente → autonome, sans script d'init externe.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.EnsureSchema(ctx, db); err != nil {
		slog.Error("schéma", "error", err)
		os.Exit(1)
	}

	repo := repository.NewNotificationRepository(db)
	hub := realtime.NewHub()
	resolver := userdir.New(cfg.UserServiceURL)
	svc := service.NewNotificationService(repo, hub, resolver)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	handler.RegisterRoutes(r, serviceName, svc, hub, cfg.JWTSecret, cfg.InternalSecret, cfg.AllowedOrigins)

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
