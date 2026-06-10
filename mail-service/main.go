package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/webdad/mail-service/internal/config"
	"github.com/webdad/mail-service/internal/handler"
	"github.com/webdad/mail-service/internal/mailer"
)

const serviceName = "mail-service"

func main() {
	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	// Le mailer choisit son transport au démarrage : SMTP si configuré,
	// sinon repli console (les e-mails sont loggés, pratique en dev).
	m := mailer.New(cfg)

	r := gin.Default()
	handler.RegisterRoutes(r, serviceName, m, cfg.InternalSecret)

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
