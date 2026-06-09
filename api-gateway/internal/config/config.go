// Package config lit la configuration du gateway depuis l'environnement.
package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config regroupe la configuration runtime du gateway.
type Config struct {
	Port           string
	GinMode        string
	AllowedOrigins []string          // origines autorisées en CORS (front)
	Services       map[string]string // préfixe de route -> URL du service cible
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars) :
//   - .env     : config propre au gateway (PORT, *_SERVICE_URL…)
//   - ../.env  : vars transverses de la racine
//
// godotenv n'écrase jamais une variable déjà présente : en stack Docker,
// les overrides compose (URLs = noms de conteneurs) restent prioritaires.
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	return &Config{
		Port:           getEnv("PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		Services: map[string]string{
			"/auth":          getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
			"/users":         getEnv("USER_SERVICE_URL", "http://localhost:8082"),
			"/profils":       getEnv("PROFIL_SERVICE_URL", "http://localhost:8083"),
			"/posts":         getEnv("POST_SERVICE_URL", "http://localhost:8084"),
			"/messages":      getEnv("MESSAGE_SERVICE_URL", "http://localhost:8085"),
			"/notifications": getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8086"),
			"/media":         getEnv("MEDIA_SERVICE_URL", "http://localhost:8087"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitCSV découpe une liste séparée par des virgules en éléments nettoyés.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
