package config

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSMTPConfigured(t *testing.T) {
	tests := []struct {
		name string
		smtp SMTP
		want bool
	}{
		{
			name: "complete",
			smtp: SMTP{Host: "smtp.example.com", User: "user", Password: "password"},
			want: true,
		},
		{
			name: "missing host",
			smtp: SMTP{User: "user", Password: "password"},
			want: false,
		},
		{
			name: "missing user",
			smtp: SMTP{Host: "smtp.example.com", Password: "password"},
			want: false,
		},
		{
			name: "missing password",
			smtp: SMTP{Host: "smtp.example.com", User: "user"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.smtp.Configured(); got != tt.want {
				t.Fatalf("Configured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadUsesEnvironmentAndDefaults(t *testing.T) {
	t.Setenv("MAIL_INTERNAL_SECRET", "internal-secret")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_USER", "user@example.com")
	t.Setenv("SMTP_PASSWORD", "app-password")
	t.Setenv("SMTP_FROM", "Breezy <no-reply@example.com>")
	t.Setenv("PORT", "")
	t.Setenv("GIN_MODE", "")
	t.Setenv("SMTP_PORT", "")

	cfg := Load()

	if cfg.Port != "8089" {
		t.Fatalf("Port = %q, want default 8089", cfg.Port)
	}
	if cfg.GinMode != "debug" {
		t.Fatalf("GinMode = %q, want default debug", cfg.GinMode)
	}
	if cfg.InternalSecret != "internal-secret" {
		t.Fatalf("InternalSecret = %q, want internal-secret", cfg.InternalSecret)
	}
	if cfg.SMTP.Port != "587" {
		t.Fatalf("SMTP.Port = %q, want default 587", cfg.SMTP.Port)
	}
	if !cfg.SMTP.Configured() {
		t.Fatal("SMTP should be configured from environment")
	}
}

func TestLoadUsesExplicitEnvironmentValues(t *testing.T) {
	t.Setenv("MAIL_INTERNAL_SECRET", "secret")
	t.Setenv("PORT", "9090")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.GinMode != "release" {
		t.Fatalf("GinMode = %q, want release", cfg.GinMode)
	}
	if cfg.SMTP.Port != "2525" {
		t.Fatalf("SMTP.Port = %q, want 2525", cfg.SMTP.Port)
	}
	if cfg.SMTP.Configured() {
		t.Fatal("SMTP should not be configured without host/user/password")
	}
}

func TestLoadExitsWhenInternalSecretIsMissing(t *testing.T) {
	if os.Getenv("MAIL_CONFIG_FATAL_HELPER") == "1" {
		_ = os.Unsetenv("MAIL_INTERNAL_SECRET")
		Load()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLoadExitsWhenInternalSecretIsMissing")
	cmd.Env = append(os.Environ(), "MAIL_CONFIG_FATAL_HELPER=1", "MAIL_INTERNAL_SECRET=")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Load() without MAIL_INTERNAL_SECRET should exit")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("cmd.Run() error = %T %v, want *exec.ExitError", err, err)
	}
	if exitErr.Success() {
		t.Fatal("helper process unexpectedly succeeded")
	}
	if !strings.Contains(string(output), "MAIL_INTERNAL_SECRET manquant") {
		t.Fatalf("output = %q, want missing secret message", string(output))
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("MAIL_TEST_VALUE", "configured")
	if got := getEnv("MAIL_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("getEnv configured = %q, want configured", got)
	}
	if got := getEnv("MAIL_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("getEnv fallback = %q, want fallback", got)
	}
}
