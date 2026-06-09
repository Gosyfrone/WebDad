package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/config"
	"github.com/webdad/message-service/internal/database"
	"github.com/webdad/message-service/internal/handler"
	"github.com/webdad/message-service/internal/notifier"
	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/repository"
	"github.com/webdad/message-service/internal/service"
)

const serviceName = "message-service"

func main() {
	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	client, err := database.ConnectMongo(cfg.MongoURI)
	if err != nil {
		log.Fatalf("[%s] connexion Mongo : %v", serviceName, err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()

	db := client.Database(cfg.MongoDB)

	// Le service applique son propre schéma (collections + validateurs + index)
	// au démarrage, de façon idempotente → autonome, sans script d'init externe.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.EnsureSchema(ctx, db); err != nil {
		log.Fatalf("[%s] schéma : %v", serviceName, err)
	}

	repo := repository.NewMessageRepository(db)
	svc := service.NewMessageService(repo)

	// Émission des mentions en message vers le notification-service (best-effort,
	// fire-and-forget). Activée uniquement si le service est configuré → le
	// message-service reste autonome sans lui.
	if cfg.NotificationURL != "" && cfg.InternalSecret != "" {
		svc.SetNotifier(notifier.New(cfg.NotificationURL, cfg.InternalSecret))
		log.Printf("[%s] notifications activées → %s", serviceName, cfg.NotificationURL)
	}

	hub := realtime.NewHub()

	r := gin.Default()
	handler.RegisterRoutes(r, serviceName, svc, hub, cfg.JWTSecret, cfg.AllowedOrigins)

	log.Printf("[%s] en écoute sur le port %s", serviceName, cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("[%s] échec du démarrage : %v", serviceName, err)
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
