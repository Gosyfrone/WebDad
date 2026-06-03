package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/config"
	"github.com/webdad/auth-service/internal/db"
	"github.com/webdad/auth-service/internal/router"
	"github.com/webdad/auth-service/internal/services"
)

const serviceName = "auth-service"

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

	auth := services.New(conn, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)

	if cfg.SeedAdmin {
		if err := auth.EnsureDefaultAdmin(cfg.SeedAdminEmail, cfg.SeedAdminPassword); err != nil {
			log.Fatalf("[%s] seed admin : %v", serviceName, err)
		}
		log.Printf("[%s] admin par défaut assuré (%s)", serviceName, cfg.SeedAdminEmail)
	}

	r := router.New(auth)

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
