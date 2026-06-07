// Package config lit la configuration du service depuis les variables
// d'environnement. Aucune valeur en dur : l'URI Mongo est construite à
// partir du .env (cf. convention décrite dans post-service/.env).
package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config regroupe la configuration runtime du service.
type Config struct {
	Port      string
	GinMode   string
	MongoURI  string
	MongoDB   string
	JWTSecret string // secret partagé (validation des tokens émis par auth)
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
		Port:      getEnv("PORT", "8084"),
		GinMode:   getEnv("GIN_MODE", "debug"),
		MongoURI:  buildMongoURI(),
		MongoDB:   getEnv("MONGO_INITDB_DATABASE", "webdad_post"),
		JWTSecret: os.Getenv("JWT_SECRET"),
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
