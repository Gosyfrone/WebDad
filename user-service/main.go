package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/client"
	"github.com/webdad/user-service/internal/config"
	"github.com/webdad/user-service/internal/db"
	"github.com/webdad/user-service/internal/repository"
	"github.com/webdad/user-service/internal/router"
	"github.com/webdad/user-service/internal/service"
)

const serviceName = "user-service"

func main() {
	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[%s] connexion DB : %v", serviceName, err)
	}
	defer func() { _ = conn.Close() }()

	if err := db.EnsureSchema(conn); err != nil {
		log.Fatalf("[%s] schéma : %v", serviceName, err)
	}

	opts := []service.Option{
		service.WithProfilClient(client.NewProfilClient(cfg.ProfilServiceURL)),
	}
	if cfg.NotificationServiceURL != "" {
		opts = append(opts, service.WithNotificationClient(client.NewNotificationClient(cfg.NotificationServiceURL, cfg.InternalSecret)))
	}
	users := service.New(repository.New(conn), cfg.UsernameCooldown, opts...)
	r := router.New(users, cfg.JWTSecret)

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
