package logging

import (
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetup_NePasPlanter(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("LOG_LEVEL", "")
	Setup("test-service")
	if slog.Default() == nil {
		t.Fatal("Setup devrait initialiser un logger non nil")
	}
}

func TestSetup_ModeRelease(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	Setup("test-service-release")
}

func TestSetup_NiveauxValides(t *testing.T) {
	for _, level := range []string{"debug", "warn", "error", "info", ""} {
		t.Run(level, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", level)
			Setup("svc")
		})
	}
}

func TestFromGin_SansUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("request_id", "req-abc")
	l := FromGin(c)
	if l == nil {
		t.Fatal("FromGin devrait retourner un logger non nil")
	}
}

func TestFromGin_AvecUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("request_id", "req-xyz")
	c.Set("user_id", "uid-123")
	l := FromGin(c)
	if l == nil {
		t.Fatal("FromGin avec user_id devrait retourner un logger non nil")
	}
}
