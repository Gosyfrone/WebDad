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
	Port          string
	GinMode       string
	DatabaseURL   string        // DSN PostgreSQL (lib/pq)
	JWTSecret     string        // secret partagé (signature + validation)
	JWTExpiry     time.Duration // durée de validité de l'access token (court)
	RefreshExpiry time.Duration // durée de validité du refresh token (long)

	// Seed admin (dev) : crée un compte admin au démarrage si activé.
	SeedAdmin         bool
	SeedAdminEmail    string
	SeedAdminPassword string

	// Intégration mail (vérification d'e-mail). MailInternalSecret/MailServiceURL
	// absents → l'envoi devient un no-op loggé (auth reste bootable seul).
	MailServiceURL     string // base URL du mail-service (POST /internal/send)
	MailInternalSecret string // secret partagé X-Internal-Secret (= .env racine)
	AppBaseURL         string // base URL du front (liens dans les e-mails)

	// AdminCreateAutoVerify : raccourci de DEV/LOCAL. Quand true, un compte créé
	// par un admin est marqué vérifié d'office (email_verified=true) → la
	// vérification d'e-mail est court-circuitée et l'utilisateur peut se connecter
	// directement avec le mot de passe temporaire. En PROD (false), il doit
	// vérifier son adresse via le lien reçu par e-mail avant de pouvoir entrer.
	AdminCreateAutoVerify bool

	// Effacement RGPD automatique des comptes bannis. URLs des services à purger
	// (vide → étape ignorée). AccountPurgeAfter : ancienneté du bannissement avant
	// purge (défaut 5 ans) ; AccountPurgeSweepInterval : période de balayage
	// (défaut 12h ; <= 0 sur l'une ou l'autre = balayage désactivé).
	UserServiceURL    string
	ProfilServiceURL  string
	PostServiceURL    string
	MessageServiceURL string
	MediaServiceURL   string

	AccountPurgeAfter         time.Duration
	AccountPurgeSweepInterval time.Duration

	// OAuth OIDC (Login with Google). Un provider sans ClientID est simplement
	// ignoré (endpoints → 404). Le redirect URI front est dérivé de
	// OAuthRedirectBaseURL : <base>/auth/callback/<provider>.
	OAuthRedirectBaseURL string
	GoogleClientID       string
	GoogleClientSecret   string
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

	cfg.MailServiceURL = getEnv("MAIL_SERVICE_URL", "http://localhost:8089")
	cfg.MailInternalSecret = os.Getenv("MAIL_INTERNAL_SECRET")
	cfg.AppBaseURL = getEnv("APP_BASE_URL", "http://localhost:3000")
	// DEV/LOCAL : court-circuite la vérification d'e-mail des comptes créés par
	// un admin (l'envoi de mail réel se fait en ligne). À laisser false en prod.
	cfg.AdminCreateAutoVerify = getEnv("ADMIN_CREATE_AUTO_VERIFY", "false") == "true"

	cfg.SeedAdmin = getEnv("SEED_DEFAULT_ADMIN", "false") == "true"
	cfg.SeedAdminEmail = getEnv("SEED_ADMIN_EMAIL", "admin@webdad.local")
	cfg.SeedAdminPassword = os.Getenv("SEED_ADMIN_PASSWORD")
	if cfg.SeedAdmin && cfg.SeedAdminPassword == "" {
		log.Fatal("[config] SEED_DEFAULT_ADMIN=true exige SEED_ADMIN_PASSWORD")
	}

	cfg.UserServiceURL = getEnv("USER_SERVICE_URL", defaultServiceURL("user-service", "8082"))
	cfg.ProfilServiceURL = getEnv("PROFIL_SERVICE_URL", defaultServiceURL("profil-service", "8083"))
	cfg.PostServiceURL = getEnv("POST_SERVICE_URL", defaultServiceURL("post-service", "8084"))
	cfg.MessageServiceURL = getEnv("MESSAGE_SERVICE_URL", defaultServiceURL("message-service", "8085"))
	cfg.MediaServiceURL = getEnv("MEDIA_SERVICE_URL", defaultServiceURL("media-service", "8087"))
	cfg.AccountPurgeAfter = parseDurationOr("ACCOUNT_PURGE_AFTER", 43800*time.Hour) // ~5 ans
	cfg.AccountPurgeSweepInterval = parseDurationOr("ACCOUNT_PURGE_SWEEP_INTERVAL", 12*time.Hour)

	cfg.OAuthRedirectBaseURL = getEnv("OAUTH_REDIRECT_BASE_URL", "http://localhost:3000")
	cfg.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	cfg.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

	return cfg
}

// parseDurationOr lit une durée Go depuis l'env, avec repli silencieux sur la
// valeur par défaut si absente ou invalide.
func parseDurationOr(key string, fallback time.Duration) time.Duration {
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

// defaultServiceURL : nom de conteneur en stack Docker (présence de /.dockerenv),
// localhost sinon.
func defaultServiceURL(serviceName, port string) string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return fmt.Sprintf("http://%s:%s", serviceName, port)
	}
	return fmt.Sprintf("http://localhost:%s", port)
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
