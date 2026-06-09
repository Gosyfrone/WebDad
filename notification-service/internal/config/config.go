// Package config lit la configuration du notification-service depuis les
// variables d'environnement. Aucune valeur en dur : l'URI Mongo est construite
// à partir du .env (même convention que post-service / message-service).
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config regroupe la configuration runtime du service.
type Config struct {
	Port           string
	GinMode        string
	MongoURI       string
	MongoDB        string
	JWTSecret      string   // secret partagé (validation des tokens émis par auth)
	AllowedOrigins []string // origines acceptées pour l'upgrade WebSocket
	// InternalSecret protège l'endpoint d'ingestion d'événements
	// (POST /internal/events) appelé par les autres services (post-service).
	// Communication serveur-à-serveur sur le réseau Docker, jamais exposée au
	// client → ne transite pas par la gateway.
	InternalSecret string
	// UserServiceURL : base du user-service, utilisée UNIQUEMENT pour résoudre
	// les mentions (@handle → user_id) — seul le user-service connaît les handles.
	UserServiceURL string
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars).
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:           getEnv("PORT", "8086"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		MongoURI:       buildMongoURI(),
		MongoDB:        getEnv("MONGO_INITDB_DATABASE", "webdad_notification"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		InternalSecret: os.Getenv("INTERNAL_EVENT_SECRET"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "http://localhost:8082"),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("[config] JWT_SECRET manquant (à définir dans le .env racine)")
	}
	if cfg.InternalSecret == "" {
		log.Fatal("[config] INTERNAL_EVENT_SECRET manquant (à définir dans le .env racine)")
	}

	return cfg
}

// buildMongoURI assemble l'URI à partir des variables d'env. MONGO_HOST est
// surchargé par le nom du conteneur en stack Docker. authSource=admin car
// l'utilisateur root est créé dans la base admin par l'image officielle mongo.
func buildMongoURI() string {
	host := getEnv("MONGO_HOST", "localhost")
	port := getEnv("MONGO_PORT", "27017")
	user := getEnv("MONGO_INITDB_ROOT_USERNAME", "")
	pass := getEnv("MONGO_INITDB_ROOT_PASSWORD", "")

	if user == "" {
		return fmt.Sprintf("mongodb://%s:%s", host, port)
	}
	return fmt.Sprintf("mongodb://%s:%s@%s:%s/?authSource=admin", user, pass, host, port)
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
