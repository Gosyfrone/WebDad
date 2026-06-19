// Package config lit la configuration du service depuis les variables
// d'environnement. Aucune valeur en dur : les identifiants MinIO viennent du
// .env du service (partagés avec le conteneur minio, cf. media-service/.env),
// le secret JWT vient du .env racine (injecté par docker-compose).
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Tailles maximales par défaut (octets), appliquées aux utilisateurs NON-admin.
// Surchargées par l'env. Alignées sur le plus petit cap d'un service de
// référence (X.com : photo 5 Mo) → 5 Mo uniforme. Les administrateurs ne sont
// PAS plafonnés (bypass au niveau du handler) : aucun cap admin à configurer.
const (
	defaultMaxImageBytes = 5 * 1024 * 1024 // 5 Mo
	defaultMaxVideoBytes = 5 * 1024 * 1024 // 5 Mo
	defaultMaxBlobBytes  = 5 * 1024 * 1024 // 5 Mo (pièces jointes E2EE chiffrées)
)

// Config regroupe la configuration runtime du service.
type Config struct {
	Port      string
	GinMode   string
	JWTSecret string // secret partagé (validation des tokens émis par auth)

	// MinIO (stockage objet). MinioEndpoint = host:port (surchargé par le nom
	// du conteneur en stack Docker). Access/Secret = identifiants root du
	// conteneur minio (mêmes vars que l'image officielle → un seul .env).
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
	MinioBucket    string

	// Caps de taille d'upload (utilisateurs non-admin), par nature de média.
	// Les administrateurs bypassent ces caps (cf. handler.Upload).
	MaxImageBytes int64
	MaxVideoBytes int64
	MaxBlobBytes  int64 // pièces jointes chiffrées E2EE (POST /media/encrypted)
}

// Load construit la config. Charge les .env best-effort (ignorés s'ils
// n'existent pas, p. ex. en conteneur où compose injecte déjà les vars) :
//   - .env     : config propre au service (PORT, MINIO_*)
//   - ../.env  : vars transverses de la racine (JWT_SECRET…)
//
// godotenv n'écrase jamais une variable déjà présente : en stack Docker,
// l'override compose (MINIO_ENDPOINT=minio:9000) reste prioritaire.
func Load() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:           getEnv("PORT", "8087"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ROOT_USER", ""),
		MinioSecretKey: getEnv("MINIO_ROOT_PASSWORD", ""),
		MinioUseSSL:    getBool("MINIO_USE_SSL", false),
		MinioBucket:    getEnv("MINIO_BUCKET", "breezy-media"),
		MaxImageBytes:  getInt64("MEDIA_MAX_IMAGE_BYTES", defaultMaxImageBytes),
		MaxVideoBytes:  getInt64("MEDIA_MAX_VIDEO_BYTES", defaultMaxVideoBytes),
		MaxBlobBytes:   getInt64("MEDIA_MAX_BLOB_BYTES", defaultMaxBlobBytes),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("[config] JWT_SECRET manquant (à définir dans le .env racine)")
	}
	if cfg.MinioAccessKey == "" || cfg.MinioSecretKey == "" {
		log.Fatal("[config] MINIO_ROOT_USER / MINIO_ROOT_PASSWORD manquants (cf. media-service/.env)")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getInt64 lit un entier depuis l'env. Valeur invalide/absente → fallback.
func getInt64(key string, fallback int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		log.Printf("[config] %s invalide (%q) : %v — valeur par défaut %d", key, raw, err, fallback)
		return fallback
	}
	return v
}

// getBool lit un booléen depuis l'env (true/1/yes). Absent → fallback.
func getBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("[config] %s invalide (%q) : %v — valeur par défaut %t", key, raw, err, fallback)
		return fallback
	}
	return v
}
