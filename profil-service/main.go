package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/profil-service/internal/client"
	"github.com/webdad/profil-service/internal/config"
	"github.com/webdad/profil-service/internal/database"
	"github.com/webdad/profil-service/internal/handler"
	"github.com/webdad/profil-service/internal/logging"
	"github.com/webdad/profil-service/internal/middleware"
	"github.com/webdad/profil-service/internal/repository"
	"github.com/webdad/profil-service/internal/service"
)

const serviceName = "profil-service"

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

	// Le service applique son propre schéma (collection + validateur + index +
	// seed admin) au démarrage, de façon idempotente → autonome, sans script
	// d'init externe.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.EnsureSchema(ctx, db); err != nil {
		slog.Error("schéma", "error", err)
		os.Exit(1)
	}

	// On logue l'URL user-service résolue : un `http://localhost:...` ici en
	// conteneur = USER_SERVICE_URL absent de l'environnement (conteneur à recréer)
	// → les appels inter-services (is-following, accept-all) échoueraient.
	slog.Info("user-service", "url", cfg.UserURL)
	profils := service.New(
		repository.NewProfilRepository(db),
		cfg.DisplayNameCooldown,
		service.WithFollowChecker(client.NewUserClient(cfg.UserURL)),
	)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	handler.RegisterRoutes(r, serviceName, profils, cfg.JWTSecret)

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
