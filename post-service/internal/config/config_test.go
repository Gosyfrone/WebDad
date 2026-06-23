package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("MONGO_HOST", "mongo")
	t.Setenv("MONGO_PORT", "27019")
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "root")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "secret")
	t.Setenv("MONGO_INITDB_DATABASE", "post_test")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("USER_SERVICE_URL", "http://users")
	t.Setenv("PROFIL_SERVICE_URL", "http://profils")
	t.Setenv("BOOKMARK_SESSION_WINDOW", "10m")
	t.Setenv("PURGE_AFTER", "100h")
	t.Setenv("PURGE_WARN_BEFORE", "10h")
	t.Setenv("PURGE_SWEEP_INTERVAL", "2h")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a, http://b")

	// Load charge des .env best-effort depuis le cwd : on se place dans un dossier
	// vide pour éviter de capter le .env du dépôt (qui écraserait nos valeurs… non,
	// godotenv n'écrase pas, mais évite tout effet de bord de fichiers absents).
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cfg := Load()
	if cfg.Port != "9999" || cfg.GinMode != "release" || cfg.MongoDB != "post_test" {
		t.Fatalf("config simple inattendue: %#v", cfg)
	}
	if cfg.MongoURI != "mongodb://root:secret@mongo:27019/?authSource=admin" {
		t.Fatalf("MongoURI = %q", cfg.MongoURI)
	}
	if cfg.JWTSecret != "jwt-secret" || cfg.UserServiceURL != "http://users" || cfg.ProfilServiceURL != "http://profils" {
		t.Fatalf("config secrets/url inattendue: %#v", cfg)
	}
	if cfg.BookmarkWindow != 10*time.Minute {
		t.Fatalf("BookmarkWindow = %s", cfg.BookmarkWindow)
	}
	if cfg.PurgeAfter != 100*time.Hour || cfg.PurgeWarnBefore != 10*time.Hour || cfg.PurgeSweepInterval != 2*time.Hour {
		t.Fatalf("purge durations inattendues: %#v", cfg)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "http://a" || cfg.AllowedOrigins[1] != "http://b" {
		t.Fatalf("AllowedOrigins = %#v", cfg.AllowedOrigins)
	}
}

func TestBuildMongoURI(t *testing.T) {
	t.Setenv("MONGO_HOST", "mongo-post")
	t.Setenv("MONGO_PORT", "27018")
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "")
	if got := buildMongoURI(); got != "mongodb://mongo-post:27018" {
		t.Fatalf("mongo URI sans auth = %q", got)
	}

	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "root")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "secret")
	if got := buildMongoURI(); got != "mongodb://root:secret@mongo-post:27018/?authSource=admin" {
		t.Fatalf("mongo URI avec auth = %q", got)
	}
}

func TestGetDuration(t *testing.T) {
	fallback := 5 * time.Minute

	t.Setenv("BOOKMARK_SESSION_WINDOW", "168h")
	if got := getDuration("BOOKMARK_SESSION_WINDOW", fallback); got != 168*time.Hour {
		t.Fatalf("duration valide = %s", got)
	}

	t.Setenv("BOOKMARK_SESSION_WINDOW", "nope")
	if got := getDuration("BOOKMARK_SESSION_WINDOW", fallback); got != fallback {
		t.Fatalf("duration invalide = %s, attendu fallback %s", got, fallback)
	}

	t.Setenv("BOOKMARK_SESSION_WINDOW", "")
	if got := getDuration("BOOKMARK_SESSION_WINDOW", fallback); got != fallback {
		t.Fatalf("duration absente = %s, attendu fallback %s", got, fallback)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("POST_TEST_KEY", "")
	if got := getEnv("POST_TEST_KEY", "fallback"); got != "fallback" {
		t.Fatalf("getEnv absent = %q", got)
	}

	t.Setenv("POST_TEST_KEY", "value")
	if got := getEnv("POST_TEST_KEY", "fallback"); got != "value" {
		t.Fatalf("getEnv present = %q", got)
	}
}

func TestDefaultServiceURL(t *testing.T) {
	// Hors conteneur (/.dockerenv absent sur le runner CI/local) → localhost.
	if got := defaultServiceURL("user-service", "8082"); got != "http://localhost:8082" {
		t.Fatalf("default local URL = %q", got)
	}
}

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"http://a", 1},
		{"http://a,http://b", 2},
		{" http://a , , http://b ", 2},
	}
	for _, tc := range cases {
		if got := len(splitCSV(tc.in)); got != tc.want {
			t.Fatalf("splitCSV(%q) = %d, attendu %d", tc.in, got, tc.want)
		}
	}
}
