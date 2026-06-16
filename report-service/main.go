package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/report-service/internal/config"
	"github.com/webdad/report-service/internal/database"
	"github.com/webdad/report-service/internal/handler"
	"github.com/webdad/report-service/internal/logging"
	"github.com/webdad/report-service/internal/middleware"
	"github.com/webdad/report-service/internal/repository"
	"github.com/webdad/report-service/internal/service"
)

const serviceName = "report-service"

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

	// Le service applique son propre schéma (collections + validateurs + index)
	// au démarrage, de façon idempotente → autonome, sans script d'init externe.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.EnsureSchema(ctx, db); err != nil {
		slog.Error("schéma", "error", err)
		os.Exit(1)
	}

	repo := repository.NewReportRepository(db)
	svc := service.NewReportService(repo)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	handler.RegisterRoutes(r, serviceName, svc, cfg.JWTSecret)

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
