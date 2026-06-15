package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/client"
	"github.com/webdad/post-service/internal/config"
	"github.com/webdad/post-service/internal/database"
	"github.com/webdad/post-service/internal/handler"
	"github.com/webdad/post-service/internal/logging"
	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/notifier"
	"github.com/webdad/post-service/internal/repository"
	"github.com/webdad/post-service/internal/service"
)

const serviceName = "post-service"

func main() {
	logging.Setup(serviceName)

	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	mongoClient, err := database.ConnectMongo(cfg.MongoURI)
	if err != nil {
		slog.Error("connexion Mongo", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(ctx)
	}()

	db := mongoClient.Database(cfg.MongoDB)

	// Le service applique son propre schéma (collections + validateurs +
	// index) au démarrage, de façon idempotente → autonome, sans script
	// d'init externe.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.EnsureSchema(ctx, db); err != nil {
		slog.Error("schéma", "error", err)
		os.Exit(1)
	}

	postRepo := repository.NewPostRepository(db)
	opts := []service.Option{
		service.WithBookmarkWindow(cfg.BookmarkWindow),
		service.WithProfilClient(client.NewProfilClient(cfg.ProfilServiceURL)),
		service.WithFollowClient(client.NewFollowClient(cfg.UserServiceURL, cfg.InternalSecret)),
		service.WithPurgeRetention(cfg.PurgeAfter, cfg.PurgeWarnBefore),
	}
	if cfg.NotificationURL != "" {
		opts = append(opts, service.WithNotifier(notifier.New(cfg.NotificationURL, cfg.InternalSecret)))
	}
	postService := service.NewPostService(postRepo, opts...)

	// Balayage RGPD des tweets masqués (préavis + purge définitive) en arrière-plan.
	sweepCtx, stopSweeper := context.WithCancel(context.Background())
	defer stopSweeper()
	go postService.RunPurgeSweeper(sweepCtx, cfg.PurgeSweepInterval)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	handler.RegisterRoutes(r, serviceName, postService, cfg.JWTSecret)

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
