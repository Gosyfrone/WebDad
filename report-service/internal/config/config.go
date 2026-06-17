// Package config lit la configuration du report-service depuis les variables
// d'environnement. Aucune valeur en dur : l'URI Mongo est construite à partir
// du .env (même convention que notification-service / post-service).
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
	AllowedOrigins []string // origines acceptées (CORS, alignées sur la gateway)
	// PostServiceURL : base du post-service, appelé en serveur-à-serveur pour
	// l'auto-masquage/démasquage des posts trop signalés. Vide → auto-masquage
	// désactivé (report-service reste autonome). InternalSecret authentifie ces
	// appels (en-tête X-Internal-Secret, même secret partagé que les notifications).
	PostServiceURL string
	InternalSecret string
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars).
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:           getEnv("PORT", "8090"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		MongoURI:       buildMongoURI(),
		MongoDB:        getEnv("MONGO_INITDB_DATABASE", "webdad_report"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		PostServiceURL: getEnv("POST_SERVICE_URL", defaultServiceURL("post-service", "8084")),
		InternalSecret: getEnv("INTERNAL_SECRET", os.Getenv("INTERNAL_EVENT_SECRET")),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("[config] JWT_SECRET manquant (à définir dans le .env racine)")
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

// defaultServiceURL construit l'URL par défaut d'un service interne : nom de
// conteneur sur le réseau Docker, localhost sinon (dev hors conteneur).
func defaultServiceURL(serviceName, port string) string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return fmt.Sprintf("http://%s:%s", serviceName, port)
	}
	return fmt.Sprintf("http://localhost:%s", port)
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
