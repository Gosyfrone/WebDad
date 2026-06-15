package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/webdad/auth-service/internal/config"
	"github.com/webdad/auth-service/internal/db"
	"github.com/webdad/auth-service/internal/eraser"
	"github.com/webdad/auth-service/internal/logging"
	"github.com/webdad/auth-service/internal/notify"
	"github.com/webdad/auth-service/internal/oauth"
	"github.com/webdad/auth-service/internal/router"
	"github.com/webdad/auth-service/internal/services"
)

const serviceName = "auth-service"

func main() {
	logging.Setup(serviceName)

	cfg := config.Load()
	gin.SetMode(ginMode(cfg.GinMode))

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connexion DB", "error", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	if err := db.EnsureSchema(conn); err != nil {
		slog.Error("schéma", "error", err)
		os.Exit(1)
	}

	// Client mail (best-effort). Secret absent → mailer nil : l'envoi devient un
	// no-op loggé et auth reste bootable seul.
	var mailer services.Mailer
	if cfg.MailInternalSecret != "" {
		mailer = notify.NewMailClient(cfg.MailServiceURL, cfg.MailInternalSecret)
		slog.Info("mail-service configuré", "url", cfg.MailServiceURL)
	} else {
		slog.Warn("MAIL_INTERNAL_SECRET absent — envoi d'e-mails désactivé (no-op)")
	}

	auth := services.New(conn, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry, mailer, cfg.AppBaseURL, cfg.MailLogoURL, cfg.AdminCreateAutoVerify)

	if cfg.SeedAdmin {
		if err := auth.EnsureDefaultAdmin(cfg.SeedAdminEmail, cfg.SeedAdminPassword); err != nil {
			slog.Error("seed admin", "error", err)
			os.Exit(1)
		}
		slog.Info("admin par défaut assuré", "email", cfg.SeedAdminEmail)
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
