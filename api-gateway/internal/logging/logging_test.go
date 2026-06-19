package logging

import (
	"log/slog"
	"testing"
)

func TestSetup_NePasPlanter(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("LOG_LEVEL", "")
	Setup("api-gateway")
	if slog.Default() == nil {
		t.Fatal("Setup devrait initialiser un logger non nil")
	}
}

func TestSetup_ModeRelease(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	Setup("api-gateway-release")
}

func TestSetup_NiveauxValides(t *testing.T) {
	for _, level := range []string{"debug", "warn", "error", "info", ""} {
		t.Run(level, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", level)
			Setup("svc")
		})
	}
}
