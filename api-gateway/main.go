package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/webdad/api-gateway/internal/config"
	"github.com/webdad/api-gateway/internal/router"
)

const serviceName = "api-gateway"

func main() {
	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	r, err := router.New(cfg)
	if err != nil {
		log.Fatalf("[%s] configuration du routeur : %v", serviceName, err)
	}

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
