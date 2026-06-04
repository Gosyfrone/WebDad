// Package config lit la configuration du service depuis les variables
// d'environnement. Aucune valeur sensible n'est codée en dur : le secret
// JWT vient du .env racine, injecté par docker-compose.
package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config regroupe toute la configuration runtime du service.
type Config struct {
	Port        string
	GinMode     string
	DatabaseURL string // DSN PostgreSQL (lib/pq)
	JWTSecret   string // secret partagé (validation des tokens émis par auth)

	// UsernameCooldown : délai minimal imposé entre deux changements de
	// username. 0 = désactivé (défaut) — le timestamp est tout de même
	// enregistré, seul le refus est inactif. Env USERNAME_CHANGE_COOLDOWN
	// au format durée Go (ex. "168h" = 7 jours).
	UsernameCooldown time.Duration
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars) :
//   - .env     : config propre au service (PORT, DB_*, POSTGRES_*)
//   - ../.env  : vars transverses de la racine (JWT_SECRET)
//
// godotenv n'écrase JAMAIS une variable déjà présente : en stack Docker,
// les valeurs injectées par compose (DB_HOST=postgres-user, JWT_SECRET...)
// restent prioritaires.
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:             getEnv("PORT", "8082"),
		GinMode:          getEnv("GIN_MODE", "debug"),
		DatabaseURL:      buildDSN(),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		UsernameCooldown: parseDuration("USERNAME_CHANGE_COOLDOWN", 0),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("[config] JWT_SECRET manquant (à définir dans le .env racine)")
	}

	return cfg
}

// buildDSN assemble la chaîne de connexion PostgreSQL à partir des
// variables d'env. DB_HOST est surchargé par le nom du conteneur en
// stack Docker (cf. docker-compose.yml).
func buildDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("POSTGRES_USER", "webdad")
	pass := getEnv("POSTGRES_PASSWORD", "")
	name := getEnv("POSTGRES_DB", "webdad_user")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, sslmode,
	)
}

// parseDuration lit une durée Go depuis l'env (ex. "168h"). Valeur invalide
// ou absente → fallback (config tolérante, pas de fatal).
func parseDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("[config] %s invalide (%q) : %v — valeur par défaut %s", key, raw, err, fallback)
		return fallback
	}
	return d
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
