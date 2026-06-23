package config

import (
	"strings"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("FOO_X", "bar")
	if getEnv("FOO_X", "def") != "bar" {
		t.Errorf("valeur présente doit être renvoyée")
	}
	if getEnv("FOO_ABSENT", "def") != "def" {
		t.Errorf("fallback attendu si absente")
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a , b ,, c ")
	want := []string{"a", "b", "c"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("splitCSV = %v, attendu %v", got, want)
	}
	if len(splitCSV("")) != 0 {
		t.Errorf("chaîne vide → liste vide")
	}
}

func TestBuildMongoURI(t *testing.T) {
	// Sans identifiants.
	t.Setenv("MONGO_HOST", "h")
	t.Setenv("MONGO_PORT", "1234")
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "")
	if got := buildMongoURI(); got != "mongodb://h:1234" {
		t.Errorf("URI sans auth = %q", got)
	}
	// Avec identifiants.
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "u")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "p")
	if got := buildMongoURI(); got != "mongodb://u:p@h:1234/?authSource=admin" {
		t.Errorf("URI avec auth = %q", got)
	}
}

func TestLoad(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("PORT", "9999")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a,http://b")
	t.Setenv("MONGO_INITDB_DATABASE", "db_test")

	cfg := Load()
	if cfg.Port != "9999" || cfg.JWTSecret != "secret" || cfg.MongoDB != "db_test" {
		t.Errorf("Load = %+v", cfg)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
}
