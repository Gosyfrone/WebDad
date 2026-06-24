// Package config lit la configuration du service depuis les variables
// d'environnement. Aucune valeur en dur : l'URI Mongo est construite à partir
// du .env (cf. convention décrite dans profil-service/.env), le secret JWT
// vient du .env racine (injecté par docker-compose).
package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config regroupe la configuration runtime du service.
type Config struct {
	Port                string
	GinMode             string
	MongoURI            string
	MongoDB             string
	JWTSecret           string // secret partagé (validation des tokens émis par auth)
	UserURL             string // URL user-service utilisée pour lire le graphe social interne
	NotificationURL     string // URL notification-service pour les événements WS internes
	InternalEventSecret string // secret partagé POST /internal/events

	// DisplayNameCooldown : délai minimal imposé entre deux changements de
	// display_name. 0 = désactivé (défaut) — le timestamp est tout de même
	// enregistré, seul le refus est inactif. Env DISPLAY_NAME_CHANGE_COOLDOWN
	// au format durée Go (ex. "168h" = 7 jours).
	DisplayNameCooldown time.Duration
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars) :
//   - .env     : config propre au service (PORT, MONGO_*)
//   - ../.env  : vars transverses de la racine (JWT_SECRET…)
//
// godotenv n'écrase jamais une variable déjà présente : en stack Docker,
// l'override compose (MONGO_HOST=mongo-profil) reste prioritaire.
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:                getEnv("PORT", "8083"),
		GinMode:             getEnv("GIN_MODE", "debug"),
		MongoURI:            buildMongoURI(),
		MongoDB:             getEnv("MONGO_INITDB_DATABASE", "webdad_profil"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		UserURL:             getEnv("USER_SERVICE_URL", defaultUserServiceURL()),
		NotificationURL:     getEnv("NOTIFICATION_SERVICE_URL", defaultNotificationServiceURL()),
		InternalEventSecret: os.Getenv("INTERNAL_EVENT_SECRET"),
		DisplayNameCooldown: parseDuration("DISPLAY_NAME_CHANGE_COOLDOWN", 0),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("[config] JWT_SECRET manquant (à définir dans le .env racine)")
	}

	return cfg
}

func defaultNotificationServiceURL() string {
	if os.Getenv("DOCKERIZED") == "true" {
		return "http://notification-service:8086"
	}
	return "http://localhost:8086"
}

func defaultUserServiceURL() string {
	if os.Getenv("DOCKERIZED") == "true" {
		return "http://user-service:8082"
	}
	return "http://localhost:8082"
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

// parseDuration lit une durée Go depuis l'env (ex. "168h"). Valeur invalide
// ou absente → fallback (pas de fatal : la config reste tolérante).
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
