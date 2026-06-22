package config

import (
	"testing"
)

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a , b ,, c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitCSV = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitCSV[%d] = %q, attendu %q", i, got[i], want[i])
		}
	}
	if len(splitCSV("")) != 0 {
		t.Error("splitCSV(\"\") devrait être vide")
	}
}

func TestGetEnv(t *testing.T) {
	if getEnv("REPORT_TEST_ABSENT_KEY", "fallback") != "fallback" {
		t.Error("getEnv devrait retourner le fallback pour une clé absente")
	}
	t.Setenv("REPORT_TEST_PRESENT_KEY", "valeur")
	if getEnv("REPORT_TEST_PRESENT_KEY", "fallback") != "valeur" {
		t.Error("getEnv devrait retourner la valeur d'environnement")
	}
}

func TestBuildMongoURI(t *testing.T) {
	// Sans utilisateur → URI simple.
	t.Setenv("MONGO_HOST", "h")
	t.Setenv("MONGO_PORT", "1234")
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "")
	if got := buildMongoURI(); got != "mongodb://h:1234" {
		t.Errorf("buildMongoURI (sans auth) = %q", got)
	}

	// Avec utilisateur → authSource=admin.
	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "root")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "pw")
	if got := buildMongoURI(); got != "mongodb://root:pw@h:1234/?authSource=admin" {
		t.Errorf("buildMongoURI (avec auth) = %q", got)
	}
}

func TestDefaultServiceURL(t *testing.T) {
	// Hors conteneur (pas de /.dockerenv) → localhost.
	got := defaultServiceURL("post-service", "8084")
	if got != "http://localhost:8084" && got != "http://post-service:8084" {
		t.Errorf("defaultServiceURL = %q, attendu localhost ou nom de service", got)
	}
}

func TestLoad_OK(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret-de-test")
	t.Setenv("PORT", "9999")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("MONGO_INITDB_DATABASE", "db_test")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a.test,http://b.test")
	t.Setenv("POST_SERVICE_URL", "http://post:8084")
	t.Setenv("INTERNAL_SECRET", "shared")

	cfg := Load()
	if cfg.JWTSecret != "secret-de-test" {
		t.Errorf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.Port != "9999" || cfg.GinMode != "release" || cfg.MongoDB != "db_test" {
		t.Errorf("config inattendue : %+v", cfg)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins = %v, attendu 2 entrées", cfg.AllowedOrigins)
	}
	if cfg.PostServiceURL != "http://post:8084" || cfg.InternalSecret != "shared" {
		t.Errorf("URLs internes inattendues : %+v", cfg)
	}
}
