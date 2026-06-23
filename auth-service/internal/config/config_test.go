package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAndHelpers(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_EXPIRY", "10m")
	t.Setenv("REFRESH_EXPIRY", "48h")
	t.Setenv("PORT", "9090")
	t.Setenv("APP_BASE_URL", "https://app.example/")
	t.Setenv("ADMIN_CREATE_AUTO_VERIFY", "true")
	t.Setenv("SEED_DEFAULT_ADMIN", "true")
	t.Setenv("SEED_ADMIN_PASSWORD", "secret")
	t.Setenv("ACCOUNT_PURGE_AFTER", "24h")
	t.Setenv("ACCOUNT_PURGE_SWEEP_INTERVAL", "bad")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("POSTGRES_USER", "alice")
	t.Setenv("POSTGRES_PASSWORD", "pw")
	t.Setenv("POSTGRES_DB", "auth")

	cfg := Load()
	if cfg.Port != "9090" || cfg.JWTExpiry != 10*time.Minute || cfg.RefreshExpiry != 48*time.Hour {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if !cfg.SeedAdmin || !cfg.AdminCreateAutoVerify || cfg.AccountPurgeAfter != 24*time.Hour {
		t.Fatalf("boolean/duration config: %#v", cfg)
	}
	if cfg.AccountPurgeSweepInterval != 12*time.Hour {
		t.Fatalf("invalid duration fallback = %s", cfg.AccountPurgeSweepInterval)
	}
	if !strings.Contains(cfg.DatabaseURL, "host=db") || !strings.Contains(cfg.DatabaseURL, "user=alice") {
		t.Fatalf("DSN = %q", cfg.DatabaseURL)
	}
	if cfg.MailLogoURL != "https://app.example/logo_breezy.png" {
		t.Fatalf("logo URL = %q", cfg.MailLogoURL)
	}

	t.Setenv("EMPTY_VALUE", "")
	if getEnv("EMPTY_VALUE", "fallback") != "fallback" {
		t.Fatal("getEnv fallback")
	}
	if parseDurationOr("EMPTY_VALUE", time.Second) != time.Second {
		t.Fatal("duration fallback")
	}
	if mustParseDuration("EMPTY_VALUE", "2s") != 2*time.Second {
		t.Fatal("must duration fallback")
	}
	if got := defaultServiceURL("users", "1234"); !strings.Contains(got, ":1234") {
		t.Fatalf("service URL = %q", got)
	}
}
