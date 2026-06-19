package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/middleware"
	"github.com/webdad/media-service/internal/storage"
)

// RegisterRoutes enregistre les routes du media-service. L'upload et la
// suppression exigent un JWT (validation locale, secret partagé) ; le download
// est PUBLIC (l'id est non devinable, et les blobs des messages sont chiffrés
// côté client → public-par-id reste sans risque). Le préfixe `/media` est
// conservé tel quel par la gateway.
func RegisterRoutes(r *gin.Engine, serviceName string, store *storage.Store, cfg *config.Config) {
	auth := middleware.JWTAuth(cfg.JWTSecret)
	h := NewMediaHandler(store, cfg)

	r.GET("/health", Health(serviceName))

	r.POST("/media", auth, h.Upload)                    // upload image/vidéo en clair (JWT)
	r.POST("/media/encrypted", auth, h.UploadEncrypted) // blob chiffré E2EE (JWT)
	r.GET("/gifs/search", auth, h.SearchGiphy)          // recherche/tendances GIFs (JWT, clé côté serveur)
	r.GET("/media/:id/:variant", h.DownloadVariant)     // lecture publique d'une variante générée
	r.GET("/media/:id", h.Download)                     // lecture publique (stream + Range)
	r.DELETE("/media/:id", auth, h.Delete)              // suppression (propriétaire/admin)
	// Effacement RGPD : purge tous les objets d'un propriétaire (admin).
	r.DELETE("/media/owners/:id", auth, middleware.AdminOnly(), h.PurgeByOwner)
}
