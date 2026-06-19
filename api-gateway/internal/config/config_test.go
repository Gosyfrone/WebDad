package config

import (
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_KEY", "valeur")
	if got := getEnv("TEST_KEY", "fallback"); got != "valeur" {
		t.Fatalf("getEnv var définie = %q, want %q", got, "valeur")
	}
	if got := getEnv("TEST_KEY_ABSENT", "fallback"); got != "fallback" {
		t.Fatalf("getEnv var absente = %q, want %q", got, "fallback")
	}
}

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"", []string{}},
		{"http://localhost:3000", []string{"http://localhost:3000"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
	}
	for _, tc := range cases {
		got := splitCSV(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("splitCSV(%q) len = %d, want %d : %v", tc.input, len(got), len(tc.want), got)
			continue
		}
		for i, v := range got {
			if v != tc.want[i] {
				t.Errorf("splitCSV(%q)[%d] = %q, want %q", tc.input, i, v, tc.want[i])
			}
		}
	}
}

func TestLoad_Defaults(t *testing.T) {
	for _, key := range []string{
		"PORT", "GIN_MODE", "CORS_ALLOWED_ORIGINS", "JWT_SECRET",
		"AUTH_SERVICE_URL", "USER_SERVICE_URL", "PROFIL_SERVICE_URL",
		"POST_SERVICE_URL", "MESSAGE_SERVICE_URL", "NOTIFICATION_SERVICE_URL",
		"MEDIA_SERVICE_URL", "REPORT_SERVICE_URL",
	} {
		t.Setenv(key, "")
	}

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.GinMode != "debug" {
		t.Errorf("GinMode = %q, want debug", cfg.GinMode)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret = %q, want vide", cfg.JWTSecret)
	}
	if cfg.Services["/auth"] != "http://localhost:8081" {
		t.Errorf("Services[/auth] = %q", cfg.Services["/auth"])
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")
	t.Setenv("JWT_SECRET", "supersecret")
	t.Setenv("AUTH_SERVICE_URL", "http://auth:8081")
	t.Setenv("MEDIA_SERVICE_URL", "http://media:8087")

	cfg := Load()

	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want 9999", cfg.Port)
	}
	if cfg.GinMode != "release" {
		t.Errorf("GinMode = %q, want release", cfg.GinMode)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins = %v, want 2 éléments", cfg.AllowedOrigins)
	}
	if cfg.JWTSecret != "supersecret" {
		t.Errorf("JWTSecret = %q, want supersecret", cfg.JWTSecret)
	}
	if cfg.Services["/auth"] != "http://auth:8081" {
		t.Errorf("Services[/auth] = %q", cfg.Services["/auth"])
	}
	if cfg.Services["/media"] != "http://media:8087" {
		t.Errorf("Services[/media] = %q", cfg.Services["/media"])
	}
	if cfg.Services["/gifs"] != "http://media:8087" {
		t.Errorf("Services[/gifs] doit pointer vers MEDIA_SERVICE_URL : %q", cfg.Services["/gifs"])
	}
}
