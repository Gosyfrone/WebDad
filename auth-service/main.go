package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/config"
	"github.com/webdad/auth-service/internal/db"
	"github.com/webdad/auth-service/internal/eraser"
	"github.com/webdad/auth-service/internal/notify"
	"github.com/webdad/auth-service/internal/oauth"
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

	// Client mail (best-effort). Secret absent → mailer nil : l'envoi devient un
	// no-op loggé et auth reste bootable seul.
	var mailer services.Mailer
	if cfg.MailInternalSecret != "" {
		mailer = notify.NewMailClient(cfg.MailServiceURL, cfg.MailInternalSecret)
		log.Printf("[%s] mail-service configuré (%s)", serviceName, cfg.MailServiceURL)
	} else {
		log.Printf("[%s] WARNING: MAIL_INTERNAL_SECRET absent — envoi d'e-mails désactivé (no-op)", serviceName)
	}

	auth := services.New(conn, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry, mailer, cfg.AppBaseURL, cfg.AdminCreateAutoVerify)

	if cfg.SeedAdmin {
		if err := auth.EnsureDefaultAdmin(cfg.SeedAdminEmail, cfg.SeedAdminPassword); err != nil {
			log.Fatalf("[%s] seed admin : %v", serviceName, err)
		}
		log.Printf("[%s] admin par défaut assuré (%s)", serviceName, cfg.SeedAdminEmail)
	}

	// Balayage RGPD des comptes bannis depuis > 5 ans (purge cross-service via un
	// token admin minté). En arrière-plan ; désactivé si rétention/intervalle nuls.
	sweepCtx, stopSweeper := context.WithCancel(context.Background())
	defer stopSweeper()
	acctEraser := eraser.New(eraser.Targets{
		User:    cfg.UserServiceURL,
		Profil:  cfg.ProfilServiceURL,
		Post:    cfg.PostServiceURL,
		Message: cfg.MessageServiceURL,
		Media:   cfg.MediaServiceURL,
	})
	go auth.RunAccountPurgeSweeper(sweepCtx, acctEraser, cfg.AccountPurgeAfter, cfg.AccountPurgeSweepInterval)

	// Providers OAuth (Login with Google). Construction paresseuse :
	// le discovery OIDC se fait au premier usage, pas au boot.
	oauthReg := oauth.NewRegistry(context.Background(), oauth.Options{
		RedirectBaseURL: cfg.OAuthRedirectBaseURL,
		Providers: map[string]oauth.Credentials{
			"google": {
				ClientID:     cfg.GoogleClientID,
				ClientSecret: cfg.GoogleClientSecret,
			},
		},
	})

	r := router.New(auth, oauthReg)

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
