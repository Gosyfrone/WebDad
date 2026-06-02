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
	DatabaseURL string        // DSN PostgreSQL (lib/pq)
	JWTSecret   string        // secret partagé (signature + validation)
	JWTExpiry   time.Duration // durée de validité des tokens

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

	expiry := getEnv("JWT_EXPIRY", "24h")
	d, err := time.ParseDuration(expiry)
	if err != nil {
		log.Fatalf("[config] JWT_EXPIRY invalide (%q) : %v", expiry, err)
	}
	cfg.JWTExpiry = d

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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
