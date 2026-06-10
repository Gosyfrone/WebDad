// Package config lit la configuration du mail-service depuis les variables
// d'environnement. Aucune valeur sensible en dur : le secret interne vient du
// .env racine, les identifiants SMTP du .env propre au service.
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// SMTP regroupe les paramètres de connexion au serveur d'envoi (Gmail).
type SMTP struct {
	Host     string
	Port     string
	User     string
	Password string // App Password Google (jamais le mot de passe du compte)
	From     string // adresse "From" affichée (défaut : User si vide)
}

// Configured indique si le transport SMTP est utilisable. À défaut, le service
// bascule sur le transport console (dev) — voir package mailer.
func (s SMTP) Configured() bool {
	return s.Host != "" && s.User != "" && s.Password != ""
}

// Config regroupe la configuration runtime du service.
type Config struct {
	Port    string
	GinMode string
	// InternalSecret protège l'endpoint POST /internal/send appelé par
	// auth-service sur le réseau Docker. Jamais exposé au client → ne transite
	// pas par la gateway.
	InternalSecret string
	SMTP           SMTP
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars) :
//   - .env     : config propre au service (PORT, SMTP_*)
//   - ../.env  : vars transverses de la racine (MAIL_INTERNAL_SECRET)
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:           getEnv("PORT", "8089"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		InternalSecret: os.Getenv("MAIL_INTERNAL_SECRET"),
		SMTP: SMTP{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     getEnv("SMTP_PORT", "587"),
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		},
	}

	if cfg.InternalSecret == "" {
		log.Fatal("[config] MAIL_INTERNAL_SECRET manquant (à définir dans le .env racine)")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
