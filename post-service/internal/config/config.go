// Package config lit la configuration du service depuis les variables
// d'environnement. Aucune valeur en dur : l'URI Mongo est construite à
// partir du .env (cf. convention décrite dans post-service/.env).
package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config regroupe la configuration runtime du service.
type Config struct {
	Port             string
	GinMode          string
	MongoURI         string
	MongoDB          string
	JWTSecret        string // secret partagé (validation des tokens émis par auth)
	UserServiceURL   string
	ProfilServiceURL string
	// NotificationURL : base du notification-service, vers lequel post-service
	// émet ses événements (like/commentaire/mention…). Vide → émission désactivée
	// (post-service reste autonome). InternalSecret authentifie ces appels.
	NotificationURL string
	InternalSecret  string
	// BookmarkWindow : fenêtre glissante de « rafale » des signets. Un clic court
	// qui suit le précédent de moins de cette durée range automatiquement dans la
	// dernière collection ; au-delà, le serveur redemande la collection. Défaut 5m,
	// configurable via BOOKMARK_SESSION_WINDOW (format durée Go, ex. « 10m »).
	BookmarkWindow time.Duration
	// PurgeAfter : durée de rétention d'un tweet masqué (corbeille de modération)
	// avant purge définitive automatique (RGPD). Défaut 5 ans. PurgeWarnBefore :
	// préavis avant la purge (notif à l'auteur + marquage « bientôt purgé »),
	// défaut 30 jours. PurgeSweepInterval : période du balayage, défaut 6h.
	// Format durée Go (PURGE_AFTER / PURGE_WARN_BEFORE / PURGE_SWEEP_INTERVAL).
	PurgeAfter         time.Duration
	PurgeWarnBefore    time.Duration
	PurgeSweepInterval time.Duration
	// AllowedOrigins : origines acceptées pour l'upgrade WebSocket du fil temps
	// réel (`/posts/ws`). Mêmes valeurs que la CORS du front. Défaut localhost:3000.
	AllowedOrigins []string
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars) :
//   - .env     : config propre au service (PORT, MONGO_*)
//   - ../.env  : vars transverses de la racine (JWT_SECRET…)
//
// godotenv n'écrase jamais une variable déjà présente : en stack Docker,
// l'override compose (MONGO_HOST=mongo-post) reste prioritaire.
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:             getEnv("PORT", "8084"),
		GinMode:          getEnv("GIN_MODE", "debug"),
		MongoURI:         buildMongoURI(),
		MongoDB:          getEnv("MONGO_INITDB_DATABASE", "webdad_post"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		UserServiceURL:   getEnv("USER_SERVICE_URL", defaultServiceURL("user-service", "8082")),
		ProfilServiceURL: getEnv("PROFIL_SERVICE_URL", defaultServiceURL("profil-service", "8083")),
		NotificationURL:  os.Getenv("NOTIFICATION_SERVICE_URL"),
		InternalSecret:   getEnv("INTERNAL_SECRET", os.Getenv("INTERNAL_EVENT_SECRET")),
		BookmarkWindow:   getDuration("BOOKMARK_SESSION_WINDOW", 5*time.Minute),
		// 5 ans ≈ 43800h ; 30 jours = 720h.
		PurgeAfter:         getDuration("PURGE_AFTER", 43800*time.Hour),
		PurgeWarnBefore:    getDuration("PURGE_WARN_BEFORE", 720*time.Hour),
		PurgeSweepInterval: getDuration("PURGE_SWEEP_INTERVAL", 6*time.Hour),
		AllowedOrigins:     splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("[config] JWT_SECRET manquant (à définir dans le .env racine)")
	}

	return cfg
}

// getDuration lit une durée Go (ex. « 5m », « 10m ») depuis l'env, avec repli.
// Une valeur invalide journalise un avertissement et retombe sur le défaut.
func getDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("[config] %s invalide (%q) : %v — repli sur %s", key, raw, err, fallback)
		return fallback
	}
	return d
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

func defaultServiceURL(serviceName, port string) string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return fmt.Sprintf("http://%s:%s", serviceName, port)
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// splitCSV découpe une liste séparée par des virgules en éliminant les blancs et
// les entrées vides (ex. « http://a, http://b » → ["http://a","http://b"]).
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
