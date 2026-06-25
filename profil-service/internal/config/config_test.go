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
	t.Setenv("MONGO_INITDB_DATABASE", "profil_test")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("USER_SERVICE_URL", "http://users")
	t.Setenv("DISPLAY_NAME_CHANGE_COOLDOWN", "24h")
	t.Setenv("DOCKERIZED", "")

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
	if cfg.Port != "9999" || cfg.GinMode != "release" || cfg.MongoDB != "profil_test" {
		t.Fatalf("config simple inattendue: %#v", cfg)
	}
	if cfg.MongoURI != "mongodb://root:secret@mongo:27019/?authSource=admin" {
		t.Fatalf("MongoURI = %q", cfg.MongoURI)
	}
	if cfg.JWTSecret != "jwt-secret" || cfg.UserURL != "http://users" {
		t.Fatalf("config secrets/url inattendue: %#v", cfg)
	}
	if cfg.DisplayNameCooldown != 24*time.Hour {
		t.Fatalf("cooldown = %s", cfg.DisplayNameCooldown)
	}
}

func TestDefaultUserServiceURL(t *testing.T) {
	t.Setenv("DOCKERIZED", "")
	if got := defaultUserServiceURL(); got != "http://localhost:8082" {
		t.Fatalf("default local URL = %q", got)
	}

	t.Setenv("DOCKERIZED", "true")
	if got := defaultUserServiceURL(); got != "http://user-service:8082" {
		t.Fatalf("default docker URL = %q", got)
	}
}

func TestDefaultNotificationServiceURL(t *testing.T) {
	t.Setenv("DOCKERIZED", "")
	if got := defaultNotificationServiceURL(); got != "http://localhost:8086" {
		t.Fatalf("default local URL = %q", got)
	}

	t.Setenv("DOCKERIZED", "true")
	if got := defaultNotificationServiceURL(); got != "http://notification-service:8086" {
		t.Fatalf("default docker URL = %q", got)
	}
}

func TestBuildMongoURI(t *testing.T) {
	t.Setenv("MONGO_HOST", "mongo-profil")
	t.Setenv("MONGO_PORT", "27018")
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "")
	if got := buildMongoURI(); got != "mongodb://mongo-profil:27018" {
		t.Fatalf("mongo URI sans auth = %q", got)
	}

	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "root")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "secret")
	if got := buildMongoURI(); got != "mongodb://root:secret@mongo-profil:27018/?authSource=admin" {
		t.Fatalf("mongo URI avec auth = %q", got)
	}
}

func TestParseDuration(t *testing.T) {
	fallback := 15 * time.Minute

	t.Setenv("DISPLAY_NAME_CHANGE_COOLDOWN", "168h")
	if got := parseDuration("DISPLAY_NAME_CHANGE_COOLDOWN", fallback); got != 168*time.Hour {
		t.Fatalf("duration valide = %s", got)
	}

	t.Setenv("DISPLAY_NAME_CHANGE_COOLDOWN", "nope")
	if got := parseDuration("DISPLAY_NAME_CHANGE_COOLDOWN", fallback); got != fallback {
		t.Fatalf("duration invalide = %s, attendu fallback %s", got, fallback)
	}

	t.Setenv("DISPLAY_NAME_CHANGE_COOLDOWN", "")
	if got := parseDuration("DISPLAY_NAME_CHANGE_COOLDOWN", fallback); got != fallback {
		t.Fatalf("duration absente = %s, attendu fallback %s", got, fallback)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("PROFIL_TEST_KEY", "")
	if got := getEnv("PROFIL_TEST_KEY", "fallback"); got != "fallback" {
		t.Fatalf("getEnv absent = %q", got)
	}

	t.Setenv("PROFIL_TEST_KEY", "value")
	if got := getEnv("PROFIL_TEST_KEY", "fallback"); got != "value" {
		t.Fatalf("getEnv present = %q", got)
	}
}
