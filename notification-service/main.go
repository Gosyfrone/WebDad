package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

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

type runtimeDependencies struct {
	connectMongo func(string) (*mongo.Client, error)
	ensureSchema func(context.Context, *mongo.Database) error
	serve        func(*gin.Engine, ...string) error
}

func defaultRuntimeDependencies() runtimeDependencies {
	return runtimeDependencies{
		connectMongo: database.ConnectMongo,
		ensureSchema: database.EnsureSchema,
		serve:        (*gin.Engine).Run,
	}
}

func main() {
	logging.Setup(serviceName)
	if err := run(config.Load(), defaultRuntimeDependencies()); err != nil {
		slog.Error("arrêt du service", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, deps runtimeDependencies) error {
	gin.SetMode(ginMode(cfg.GinMode))

	client, err := deps.connectMongo(cfg.MongoURI)
	if err != nil {
		return fmt.Errorf("connexion Mongo : %w", err)
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
	if err := deps.ensureSchema(ctx, db); err != nil {
		return fmt.Errorf("schéma : %w", err)
	}

	repo := repository.NewNotificationRepository(db)
	hub := realtime.NewHub()
	resolver := userdir.New(cfg.UserServiceURL)
	svc := service.NewNotificationService(repo, hub, resolver)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	handler.RegisterRoutes(r, serviceName, svc, hub, cfg.JWTSecret, cfg.InternalSecret, cfg.AllowedOrigins)

	slog.Info("en écoute", "port", cfg.Port)
	if err := deps.serve(r, ":"+cfg.Port); err != nil {
		return fmt.Errorf("démarrage HTTP : %w", err)
	}
	return nil
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
