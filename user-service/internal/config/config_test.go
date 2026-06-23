package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("FOO_TEST_KEY", "value")
	if got := getEnv("FOO_TEST_KEY", "fallback"); got != "value" {
		t.Fatalf("getEnv = %q, attendu value", got)
	}
	if got := getEnv("ABSENT_TEST_KEY", "fallback"); got != "fallback" {
		t.Fatalf("getEnv(absent) = %q, attendu fallback", got)
	}
}

func TestParseDuration(t *testing.T) {
	t.Setenv("DUR_KEY", "168h")
	if got := parseDuration("DUR_KEY", 0); got != 168*time.Hour {
		t.Fatalf("parseDuration = %v, attendu 168h", got)
	}
	// Absente → fallback.
	if got := parseDuration("DUR_ABSENT", 5*time.Second); got != 5*time.Second {
		t.Fatalf("parseDuration(absent) = %v, attendu 5s", got)
	}
	// Invalide → fallback (et log).
	t.Setenv("DUR_BAD", "pas-une-durée")
	if got := parseDuration("DUR_BAD", time.Minute); got != time.Minute {
		t.Fatalf("parseDuration(invalide) = %v, attendu 1m", got)
	}
}

func TestBuildDSN(t *testing.T) {
	t.Setenv("DB_HOST", "db.example")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "mydb")
	t.Setenv("DB_SSLMODE", "require")

	dsn := buildDSN()
	for _, want := range []string{"host=db.example", "port=6543", "user=u", "password=p", "dbname=mydb", "sslmode=require"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("buildDSN = %q, manque %q", dsn, want)
		}
	}
}

func TestDefaultServiceURL(t *testing.T) {
	// Hors conteneur (/.dockerenv absent en environnement de test) → localhost.
	if _, err := os.Stat("/.dockerenv"); err == nil {
		t.Skip("/.dockerenv présent — branche conteneur, non testée ici")
	}
	got := defaultServiceURL("profil-service", "8083")
	if got != "http://localhost:8083" {
		t.Fatalf("defaultServiceURL = %q, attendu http://localhost:8083", got)
	}
}

func TestLoad(t *testing.T) {
	// Isolation : on travaille dans un répertoire temporaire sans .env, pour ne
	// pas charger un fichier du dépôt.
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir : %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Setenv("JWT_SECRET", "secret-de-test")
	t.Setenv("PORT", "9999")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("USERNAME_CHANGE_COOLDOWN", "24h")
	t.Setenv("PROFIL_SERVICE_URL", "http://profil:1")
	t.Setenv("NOTIFICATION_SERVICE_URL", "http://notif:2")
	t.Setenv("INTERNAL_SECRET", "isecret")

	cfg := Load()
	if cfg.JWTSecret != "secret-de-test" {
		t.Fatalf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.Port != "9999" || cfg.GinMode != "release" {
		t.Fatalf("Port/GinMode = %q/%q", cfg.Port, cfg.GinMode)
	}
	if cfg.UsernameCooldown != 24*time.Hour {
		t.Fatalf("UsernameCooldown = %v", cfg.UsernameCooldown)
	}
	if cfg.ProfilServiceURL != "http://profil:1" || cfg.NotificationServiceURL != "http://notif:2" {
		t.Fatalf("URLs = %q / %q", cfg.ProfilServiceURL, cfg.NotificationServiceURL)
	}
	if cfg.InternalSecret != "isecret" {
		t.Fatalf("InternalSecret = %q", cfg.InternalSecret)
	}
	if !strings.Contains(cfg.DatabaseURL, "host=") {
		t.Fatalf("DatabaseURL inattendu : %q", cfg.DatabaseURL)
	}
}

func TestLoad_InternalSecretFallback(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Setenv("JWT_SECRET", "s")
	t.Setenv("INTERNAL_SECRET", "") // vide → repli sur INTERNAL_EVENT_SECRET
	t.Setenv("INTERNAL_EVENT_SECRET", "event-secret")

	cfg := Load()
	if cfg.InternalSecret != "event-secret" {
		t.Fatalf("InternalSecret fallback = %q, attendu event-secret", cfg.InternalSecret)
	}
}
