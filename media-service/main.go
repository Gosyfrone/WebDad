package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/handler"
	"github.com/webdad/media-service/internal/storage"
)

const serviceName = "media-service"

func main() {
	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	// Le service garantit l'existence de son bucket au démarrage (idempotent) →
	// autonome, sans provisioning externe. MinIO est sa « base ».
	store, err := storage.New(
		cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey,
		cfg.MinioUseSSL, cfg.MinioBucket,
	)
	if err != nil {
		log.Fatalf("[%s] connexion MinIO : %v", serviceName, err)
	}

	r := gin.Default()
	// Les uploads peuvent atteindre la taille d'une vidéo : on laisse Gin
	// streamer plutôt que bufferiser tout le multipart en mémoire.
	r.MaxMultipartMemory = 8 << 20 // 8 Mo de buffer mémoire, le reste sur disque temp
	handler.RegisterRoutes(r, serviceName, store, cfg)

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
