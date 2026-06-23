package config

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "jwt-test")
	t.Setenv("INTERNAL_EVENT_SECRET", "internal-test")
}

func TestLoad(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("PORT", "9999")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("MONGO_HOST", "mongo.test")
	t.Setenv("MONGO_PORT", "27099")
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "root")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "password")
	t.Setenv("MONGO_INITDB_DATABASE", "notifications_test")
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://one.test, ,https://two.test ")
	t.Setenv("USER_SERVICE_URL", "http://users.test")

	cfg := Load()
	if cfg.Port != "9999" || cfg.GinMode != "release" || cfg.MongoDB != "notifications_test" {
		t.Fatalf("config de base = %#v", cfg)
	}
	if cfg.MongoURI != "mongodb://root:password@mongo.test:27099/?authSource=admin" {
		t.Fatalf("MongoURI = %q", cfg.MongoURI)
	}
	if !reflect.DeepEqual(cfg.AllowedOrigins, []string{"https://one.test", "https://two.test"}) {
		t.Fatalf("AllowedOrigins = %#v", cfg.AllowedOrigins)
	}
	if cfg.JWTSecret != "jwt-test" || cfg.InternalSecret != "internal-test" || cfg.UserServiceURL != "http://users.test" {
		t.Fatalf("config services = %#v", cfg)
	}
}

func TestDefaultsAndHelpers(t *testing.T) {
	setRequiredEnvironment(t)
	for _, key := range []string{
		"PORT", "GIN_MODE", "MONGO_HOST", "MONGO_PORT", "MONGO_INITDB_ROOT_USERNAME",
		"MONGO_INITDB_ROOT_PASSWORD", "MONGO_INITDB_DATABASE", "CORS_ALLOWED_ORIGINS", "USER_SERVICE_URL",
	} {
		t.Setenv(key, "")
	}
	cfg := Load()
	if cfg.Port != "8086" || cfg.GinMode != "debug" || cfg.MongoURI != "mongodb://localhost:27017" {
		t.Fatalf("defaults = %#v", cfg)
	}
	if got := getEnv("UNSET_NOTIFICATION_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("getEnv fallback = %q", got)
	}
	t.Setenv("SET_NOTIFICATION_TEST_VALUE", "value")
	if got := getEnv("SET_NOTIFICATION_TEST_VALUE", "fallback"); got != "value" {
		t.Fatalf("getEnv value = %q", got)
	}
	if got := splitCSV(" , a, b ,, "); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("splitCSV = %#v", got)
	}
}

func TestLoadMissingRequiredSecrets(t *testing.T) {
	if os.Getenv("CONFIG_FATAL_HELPER") != "" {
		_ = Load()
		return
	}

	tests := []struct {
		name     string
		jwt      string
		internal string
		message  string
	}{
		{"jwt", "", "internal", "JWT_SECRET manquant"},
		{"internal", "jwt", "", "INTERNAL_EVENT_SECRET manquant"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestLoadMissingRequiredSecrets")
			cmd.Dir = t.TempDir()
			cmd.Env = append(os.Environ(),
				"CONFIG_FATAL_HELPER=1",
				"JWT_SECRET="+tt.jwt,
				"INTERNAL_EVENT_SECRET="+tt.internal,
			)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("Load devait terminer le sous-processus")
			}
			if !strings.Contains(string(output), tt.message) {
				t.Fatalf("sortie = %q, attendu %q", output, tt.message)
			}
		})
	}
}
