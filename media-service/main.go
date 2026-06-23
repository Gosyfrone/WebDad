package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/handler"
	"github.com/webdad/media-service/internal/logging"
	"github.com/webdad/media-service/internal/middleware"
	"github.com/webdad/media-service/internal/storage"
)

const serviceName = "media-service"

func main() {
	logging.Setup(serviceName)
	cfg := config.Load()
	if err := run(cfg, storage.New, func(r *gin.Engine, address string) error { return r.Run(address) }); err != nil {
		slog.Error("arrêt du service", "error", err)
		os.Exit(1)
	}
}

type storeFactory func(string, string, string, bool, string) (*storage.Store, error)

func run(cfg *config.Config, newStore storeFactory, serve func(*gin.Engine, string) error) error {
	gin.SetMode(ginMode(cfg.GinMode))

	// Le service garantit l'existence de son bucket au démarrage (idempotent) →
	// autonome, sans provisioning externe. MinIO est sa « base ».
	store, err := newStore(
		cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey,
		cfg.MinioUseSSL, cfg.MinioBucket,
	)
	if err != nil {
		return fmt.Errorf("connexion MinIO: %w", err)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.RequestLogger())
	// Les uploads peuvent atteindre la taille d'une vidéo : on laisse Gin
	// streamer plutôt que bufferiser tout le multipart en mémoire.
	r.MaxMultipartMemory = 8 << 20 // 8 Mo de buffer mémoire, le reste sur disque temp
	handler.RegisterRoutes(r, serviceName, store, cfg)

	slog.Info("en écoute", "port", cfg.Port)
	if err := serve(r, ":"+cfg.Port); err != nil {
		return fmt.Errorf("échec du démarrage: %w", err)
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
