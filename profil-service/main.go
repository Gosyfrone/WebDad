package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/profil-service/internal/config"
	"github.com/webdad/profil-service/internal/database"
	"github.com/webdad/profil-service/internal/handler"
	"github.com/webdad/profil-service/internal/repository"
	"github.com/webdad/profil-service/internal/service"
)

const serviceName = "profil-service"

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

	// Le service applique son propre schéma (collection + validateur + index +
	// seed admin) au démarrage, de façon idempotente → autonome, sans script
	// d'init externe.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.EnsureSchema(ctx, db); err != nil {
		log.Fatalf("[%s] schéma : %v", serviceName, err)
	}

	profils := service.New(repository.NewProfilRepository(db), cfg.DisplayNameCooldown)

	r := gin.Default()
	handler.RegisterRoutes(r, serviceName, profils, cfg.JWTSecret)

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
