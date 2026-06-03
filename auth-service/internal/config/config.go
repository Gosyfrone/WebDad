// Package config lit la configuration du service depuis les variables
// d'environnement. Aucune valeur sensible n'est codée en dur : les secrets
// (JWT_SECRET) viennent du .env racine, injecté par docker-compose.
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
	DatabaseURL   string        // DSN PostgreSQL (lib/pq)
	JWTSecret     string        // secret partagé (signature + validation)
	JWTExpiry     time.Duration // durée de validité de l'access token (court)
	RefreshExpiry time.Duration // durée de validité du refresh token (long)

	// Seed admin (dev) : crée un compte admin au démarrage si activé.
	SeedAdmin         bool
	SeedAdminEmail    string
	SeedAdminPassword string
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars) :
//   - .env     : config propre au service (PORT, DB_*, POSTGRES_*)
//   - ../.env  : vars transverses de la racine (JWT_SECRET, JWT_EXPIRY)
//
// godotenv n'écrase JAMAIS une variable déjà présente : en stack Docker,
// les valeurs injectées par compose (DB_HOST=postgres-auth, JWT_SECRET...)
// restent prioritaires.
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:        getEnv("PORT", "8081"),
		GinMode:     getEnv("GIN_MODE", "debug"),
		DatabaseURL: buildDSN(),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("[config] JWT_SECRET manquant (à définir dans le .env racine)")
	}

	cfg.JWTExpiry = mustParseDuration("JWT_EXPIRY", "15m")
	cfg.RefreshExpiry = mustParseDuration("REFRESH_EXPIRY", "24h")

	cfg.SeedAdmin = getEnv("SEED_DEFAULT_ADMIN", "false") == "true"
	cfg.SeedAdminEmail = getEnv("SEED_ADMIN_EMAIL", "admin@webdad.local")
	cfg.SeedAdminPassword = os.Getenv("SEED_ADMIN_PASSWORD")
	if cfg.SeedAdmin && cfg.SeedAdminPassword == "" {
		log.Fatal("[config] SEED_DEFAULT_ADMIN=true exige SEED_ADMIN_PASSWORD")
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
	name := getEnv("POSTGRES_DB", "webdad_auth")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, sslmode,
	)
}

// mustParseDuration lit une durée depuis l'env (avec repli) et arrête le
// service si la valeur est mal formée (config invalide = échec au boot).
func mustParseDuration(key, fallback string) time.Duration {
	raw := getEnv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Fatalf("[config] %s invalide (%q) : %v", key, raw, err)
	}
	return d
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
